package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"anti-fraud/internal/bloom"
	"anti-fraud/internal/challenge"
	"anti-fraud/internal/headercheck"
	"anti-fraud/internal/logger"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type ClickPayload struct {
	IP             string `json:"ip"`
	UserAgent      string `json:"user_agent"`
	CampaignID     string `json:"campaign_id"`
	Timestamp      int64  `json:"timestamp"`
	ChallengeID    string `json:"challenge_id"`
	ChallengeToken string `json:"challenge_token"`
}

var (
	rdb            *redis.Client
	db             *sql.DB
	ipFilter       *bloom.IPFilter
	batchLogger    *logger.BatchLogger
	challengeStore challenge.Store
	ctx            = context.Background()
	maxRate        = 5

	// requireChallenge / requireHeaderCheck are package-level (not const)
	// specifically so existing tests that POST directly to /v1/click
	// without a challenge_id/challenge_token can flip these to false in
	// TestMain or per-test, instead of needing to solve a challenge in
	// every single existing test case. Production defaults to true for
	// both — see loadFeatureToggles().
	requireChallenge  = true
	requireHeaderCheck = true
)

func main() {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis storage")

	// Reuse the same Redis connection pool for challenge storage — no
	// second client needed.
	challengeStore = &challenge.RedisStore{Client: rdb}

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "antifraud")
	dbPassword := getEnv("DB_PASSWORD", "antifraud123")
	dbName := getEnv("DB_NAME", "analytics")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB connection: %v", err)
	}

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "80"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "20"))
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL storage")

	blacklistPath := getEnv("BLACKLIST_PATH", "./deployments/blacklists/dirty_ips.txt")
	ipFilter, err = bloom.NewIPFilter(blacklistPath)
	if err != nil {
		log.Fatalf("Failed to initialize Bloom Filter: %v", err)
	}

	batchSize, _ := strconv.Atoi(getEnv("DB_BATCH_SIZE", "1000"))
	flushIntervalMs, _ := strconv.Atoi(getEnv("DB_BATCH_FLUSH_MS", "500"))

	batchLogger = logger.NewBatchLogger(db, batchSize, flushIntervalMs)
	batchLogger.Start(ctx)
	log.Println("Asynchronous Batch Logger started")

	loadFeatureToggles()

	http.HandleFunc("/v1/click", handleClick)
	http.HandleFunc("/v1/challenge", handleChallenge)

	log.Println("Core Engine API Gateway started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// loadFeatureToggles lets the two new checks be disabled via env vars
// without a rebuild — useful for a staged rollout, and for CI if you'd
// rather keep the older tests untouched than update them all at once.
// Both default to true (enabled) in production.
func loadFeatureToggles() {
	if v, exists := os.LookupEnv("REQUIRE_JS_CHALLENGE"); exists {
		if b, err := strconv.ParseBool(v); err == nil {
			requireChallenge = b
		}
	}
	if v, exists := os.LookupEnv("REQUIRE_HEADER_CHECK"); exists {
		if b, err := strconv.ParseBool(v); err == nil {
			requireHeaderCheck = b
		}
	}
	log.Printf("Feature toggles: REQUIRE_JS_CHALLENGE=%v REQUIRE_HEADER_CHECK=%v", requireChallenge, requireHeaderCheck)
}

// parses and checks if the given user agent falls into known script signatures
func isSuspiciousUserAgent(ua string) bool {
	if ua == "" {
		return true
	}
	uaLower := strings.ToLower(ua)
	botTokens := []string{"curl", "python-requests", "go-http-client", "bot", "spider", "wget"}

	for _, token := range botTokens {
		if strings.Contains(uaLower, token) {
			return true
		}
	}
	return false
}

// handleChallenge issues a fresh nonce/challenge_id pair for GET /v1/challenge.
// The frontend clicker page fetches this before every click and solves it
// client-side (SHA-256 of nonce+salt) via crypto.subtle — see
// deployments/nginx/clicker for the matching JS.
func handleChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ch, err := challenge.Issue(r.Context(), challengeStore)
	if err != nil {
		log.Printf("Failed to issue challenge: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(ch)
}

func handleClick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getClientIP(r)
	ua := r.UserAgent()

	var payload ClickPayload
	campaignID := "unknown"

	if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
		if payload.CampaignID != "" {
			campaignID = payload.CampaignID
		}
		if payload.UserAgent != "" {
			ua = payload.UserAgent
		}
	}

	w.Header().Set("Content-Type", "application/json")

	// 1) intercept automated bot profiles before checking Bloom filters or redis counters
	clickSource := r.Header.Get("X-Click-Source")
	if clickSource == "automated" || isSuspiciousUserAgent(ua) {
		if batchLogger != nil {
			batchLogger.LogAsync(logger.ClickLog{
				IP:         ip,
				CampaignID: campaignID,
				UserAgent:  ua,
				IsBot:      true,
				Reason:     "suspicious_agent",
			})
		}
		w.WriteHeader(http.StatusOK) // allow request parsing to terminate smoothly while flagging telemetry
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "flagged",
			"message": "Click accepted for validation analysis pipeline",
		})
		return
	}

	// 2) JS-execution challenge: requires the client to have called
	// GET /v1/challenge and solved it client-side. Curl/python-requests/
	// axios/plain net/http calls that don't know about this flow fail here
	// with no_js_challenge, regardless of what UA string they send.
	if requireChallenge {
		if err := challenge.Validate(r.Context(), challengeStore, payload.ChallengeID, payload.ChallengeToken); err != nil {
			reason := "no_js_challenge"
			switch err {
			case challenge.ErrTooFast:
				reason = "challenge_too_fast"
			case challenge.ErrMismatch:
				reason = "challenge_mismatch"
			}
			if batchLogger != nil {
				batchLogger.LogAsync(logger.ClickLog{
					IP:         ip,
					CampaignID: campaignID,
					UserAgent:  ua,
					IsBot:      true,
					Reason:     reason,
				})
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "flagged",
				"message": "Click accepted for validation analysis pipeline",
			})
			return
		}
	}

	// 3) header-consistency heuristic: catches requests that spoof a
	// browser User-Agent (so they pass step 1) but don't replicate the
	// rest of what a real browser sends (Accept-Language, Accept-Encoding,
	// Sec-Fetch-*, Client Hints).
	if requireHeaderCheck {
		if hc := headercheck.Score(r); hc.IsSuspicious() {
			if batchLogger != nil {
				batchLogger.LogAsync(logger.ClickLog{
					IP:         ip,
					CampaignID: campaignID,
					UserAgent:  ua,
					IsBot:      true,
					Reason:     "suspicious_headers",
				})
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "flagged",
				"message": "Click accepted for validation analysis pipeline",
			})
			return
		}
	}

	// 4) static blacklist filter (bloom filter)
	if ipFilter != nil && ipFilter.IsBlacklisted(ip) {
		if batchLogger != nil {
			batchLogger.LogAsync(logger.ClickLog{
				IP:         ip,
				CampaignID: campaignID,
				UserAgent:  ua,
				IsBot:      true,
				Reason:     "static_blacklist",
			})
		}
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Blocked by static blacklist."})
		return
	}

	// 5) sliding window frequency threshold evaluation
	key := fmt.Sprintf("rate:%s", ip)
	currentRequests, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("Redis error: %v", err)
	} else {
		if _, expErr := rdb.ExpireNX(ctx, key, time.Second).Result(); expErr != nil {
			log.Printf("Redis ExpireNX error: %v", expErr)
		}

		if int(currentRequests) > maxRate {
			if batchLogger != nil {
				batchLogger.LogAsync(logger.ClickLog{
					IP:         ip,
					CampaignID: campaignID,
					UserAgent:  ua,
					IsBot:      true,
					Reason:     "rate_limit_exceeded",
				})
			}
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Too many requests. Real-time anti-fraud trigger."})
			return
		}
	}

	// 6) record safe validated interaction
	if batchLogger != nil {
		batchLogger.LogAsync(logger.ClickLog{
			IP:         ip,
			CampaignID: campaignID,
			UserAgent:  ua,
			IsBot:      false,
			Reason:     "allowed",
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Click registered, routing to verification queue",
	})
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
