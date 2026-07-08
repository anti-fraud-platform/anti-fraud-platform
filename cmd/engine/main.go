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
	"sync"
	"time"

	"anti-fraud/internal/challenge"
	"anti-fraud/internal/dbschema"
	"anti-fraud/internal/geoiputil"
	"anti-fraud/internal/geopolicy"
	"anti-fraud/internal/headercheck"
	"anti-fraud/internal/logger"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"crypto/sha256"
	"encoding/hex"
	"strings"
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
	batchLogger    *logger.BatchLogger
	challengeStore challenge.Store
	geoResolver    *geoiputil.Resolver
	geoPolicy      geopolicy.Config
	ctx            = context.Background()
	maxRate        = 5
	bgTasks        sync.WaitGroup

	// requireChallenge / requireHeaderCheck are package-level (not const)
	// specifically so existing tests that POST directly to /v1/click
	// without a challenge_id/challenge_token can flip these to false in
	// TestMain or per-test, instead of needing to solve a challenge in
	// every single existing test case. Production defaults to true for
	// both — see loadFeatureToggles().
	requireChallenge   = true
	requireHeaderCheck = true
	// Tier 2: risk threshold and dynamic blacklist settings
	riskThreshold         = 3                      // score >= this => blocked as "risk_score_exceeded"
	dynBlacklistThreshold = 5                      // 5+ flagged hits in window
	dynBlacklistWindow    = time.Hour              // rolling window for auto-promotion
	dynBlacklistSetKey    = "af:dynamic_blacklist" // Redis set of permanently blocked IPs
	rateKeyPrefix         = "rate:fp:"             // new prefix for fingerprint-based rate limiting
)

// healthHandler returns the health status of Redis and PostgreSQL.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{}
	// Redis
	if err := rdb.Ping(ctx).Err(); err != nil {
		status["redis"] = "unhealthy"
	} else {
		status["redis"] = "healthy"
	}
	// PostgreSQL
	if err := db.Ping(); err != nil {
		status["postgres"] = "unhealthy"
	} else {
		status["postgres"] = "healthy"
	}
	status["geoip_loaded"] = geoResolver != nil && geoResolver.HasAny()
	status["geoip_policy_enabled"] = geoPolicy.Enabled()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func main() {
	redisOptions, err := loadRedisOptions()
	if err != nil {
		log.Fatalf("Failed to configure Redis client: %v", err)
	}
	rdb = redis.NewClient(redisOptions)
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

	if err := dbschema.Apply(db); err != nil {
		log.Fatalf("Failed to apply PostgreSQL schema: %v", err)
	}
	log.Println("PostgreSQL schema is up to date")
	
	var errs []error
	geoResolver, errs = geoiputil.OpenBestEffort(geoiputil.PathsFromEnv())
	for _, openErr := range errs {
		log.Printf("Failed to open GeoIP database: %v", openErr)
	}
	if geoResolver == nil || !geoResolver.HasAny() {
		log.Fatal("Failed to initialize GeoIP resolver: no readable .mmdb database was loaded")
	}

	geoPolicy, err = geopolicy.FromEnv()
	if err != nil {
		log.Fatalf("Failed to parse GeoIP policy config: %v", err)
	}
	if !geoPolicy.Enabled() {
		log.Fatal("GeoIP-only mode requires at least one GEOIP_BLOCKED_* rule")
	}
	log.Printf("GeoIP policy loaded: %s", geoPolicy.Summary())

	batchSize, _ := strconv.Atoi(getEnv("DB_BATCH_SIZE", "1000"))
	flushIntervalMs, _ := strconv.Atoi(getEnv("DB_BATCH_FLUSH_MS", "500"))

	batchLogger = logger.NewBatchLoggerWithResolver(db, batchSize, flushIntervalMs, geoResolver)
	batchLogger.Start(ctx)
	log.Println("Asynchronous Batch Logger started")

	loadFeatureToggles()

	http.HandleFunc("/v1/click", handleClick)
	http.HandleFunc("/v1/challenge", handleChallenge)

	log.Println("Core Engine API Gateway started on :8080")
	http.HandleFunc("/health", healthHandler)
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

func loadRedisOptions() (*redis.Options, error) {
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		return redis.ParseURL(redisURL)
	}

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	username := os.Getenv("REDIS_USERNAME")
	if username == "" {
		username = os.Getenv("REDIS_USER")
	}
	if username == "" {
		username = os.Getenv("REDISUSER")
	}

	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		password = os.Getenv("REDISPASSWORD")
	}

	return &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Username: username,
		Password: password,
	}, nil
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

	// ---------- Hard checks (immediate blocking) ----------

	// 1) Dynamic blacklist (Tier 2)
	if isDynamicBlacklisted(ip) {
		if batchLogger != nil {
			batchLogger.LogAsync(logger.ClickLog{
				IP:         ip,
				CampaignID: campaignID,
				UserAgent:  ua,
				IsBot:      true,
				Reason:     "dynamic_blacklist",
			})
		}
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Blocked by dynamic blacklist."})
		return
	}

	// 2) GeoIP / ASN policy
	lookup := geoiputil.LookupResult{IP: ip}
	if geoResolver != nil {
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			lookup = geoResolver.Lookup(parsedIP)
		}
	}
	if match := geoPolicy.Evaluate(lookup); match.Blocked {
		if batchLogger != nil {
			batchLogger.LogAsync(logger.ClickLog{
				IP:         ip,
				CampaignID: campaignID,
				UserAgent:  ua,
				IsBot:      true,
				Reason:     match.Reason,
				Country:    lookup.CountryISO,
				City:       lookup.CityName,
				ASNNumber:  lookup.ASNNumber,
				ASNOrg:     lookup.ASNOrg,
			})
		}
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Blocked by GeoIP / ASN policy."})
		return
	}

	// 3) Rate limiter – now fingerprint-based (IP + UA + headers)
	fp := fingerprint(r)
	key := fmt.Sprintf("%s%s", rateKeyPrefix, fp)
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

	// ---------- Soft checks (accumulate risk score) ----------

	riskScore := 0
	riskReasons := []string{}
	clickSource := r.Header.Get("X-Click-Source")

	// a) User-Agent check
	if clickSource == "automated" || isSuspiciousUserAgent(ua) {
		riskScore += 2
		riskReasons = append(riskReasons, "suspicious_agent")
	}

	// b) JS challenge
	if requireChallenge {
		if err := challenge.Validate(r.Context(), challengeStore, payload.ChallengeID, payload.ChallengeToken); err != nil {
			switch err {
			case challenge.ErrNotFound:
				riskScore += 3
				riskReasons = append(riskReasons, "no_js_challenge")
			case challenge.ErrTooFast:
				riskScore += 3
				riskReasons = append(riskReasons, "challenge_too_fast")
			case challenge.ErrMismatch:
				riskScore += 3
				riskReasons = append(riskReasons, "challenge_mismatch")
			}
		}
	}

	// c) Header heuristic
	if requireHeaderCheck {
		hc := headercheck.Score(r)
		if hc.IsSuspicious() {
			riskScore += 2
			riskReasons = append(riskReasons, "suspicious_headers")
		}
	}

	// ---------- Final decision based on risk score ----------

	finalReason := "allowed"
	isBot := false
	if riskScore >= riskThreshold {
		isBot = true
		// If only one check fired, use that specific reason for better analytics breakdown.
		if len(riskReasons) == 1 {
			finalReason = riskReasons[0]
		} else {
			finalReason = "risk_score_exceeded"
		}
		// Asynchronously increment dynamic blacklist counter for this IP
		bgTasks.Add(1)
		go func(ip string) {
			defer bgTasks.Done()
			incrementDynamicBlacklistCounter(ip)
		}(ip)
	}

	// ---------- Log the click ----------
	if batchLogger != nil {
		batchLogger.LogAsync(logger.ClickLog{
			IP:          ip,
			CampaignID:  campaignID,
			UserAgent:   ua,
			IsBot:       isBot,
			Reason:      finalReason,
			RiskScore:   riskScore,
			RiskReasons: strings.Join(riskReasons, ","),
		})
	}

	// ---------- Response ----------
	if isBot {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "flagged",
			"message": "Click accepted for validation analysis pipeline",
		})
	} else {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Click registered, routing to verification queue",
		})
	}
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

// fingerprint creates a unique key for rate limiting based on IP and request headers.
// This prevents IP rotation attacks – a bot that changes IP but keeps the same UA/headers
// will still be rate-limited.
func fingerprint(r *http.Request) string {
	ip := getClientIP(r)
	ua := r.UserAgent()
	acceptLang := r.Header.Get("Accept-Language")
	acceptEnc := r.Header.Get("Accept-Encoding")
	raw := strings.Join([]string{ip, ua, acceptLang, acceptEnc}, "|")
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// isDynamicBlacklisted checks if IP is in the dynamic blacklist (Redis set).
func isDynamicBlacklisted(ip string) bool {
	ok, err := rdb.SIsMember(ctx, dynBlacklistSetKey, ip).Result()
	if err != nil {
		log.Printf("Redis SIsMember error: %v", err)
		return false
	}
	return ok
}

// promoteToDynamicBlacklist adds an IP to the persistent dynamic blacklist.
func promoteToDynamicBlacklist(ip string, client *redis.Client) {
	if client == nil {
		return
	}
	if err := client.SAdd(ctx, dynBlacklistSetKey, ip).Err(); err != nil {
		log.Printf("Failed to add IP to dynamic blacklist: %v", err)
		return
	}
	log.Printf("IP %s promoted to dynamic blacklist", ip)
}

// incrementDynamicBlacklistCounter increments a per-IP counter and promotes if threshold is reached.
// Called asynchronously after a flagged (risk-scored) request.
func incrementDynamicBlacklistCounter(ip string) {
	// Copy globals to locals to avoid data race with test cleanup.
	rdbLocal := rdb
	ctxLocal := ctx
	threshold := dynBlacklistThreshold

	if rdbLocal == nil || threshold > 100 {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered in incrementDynamicBlacklistCounter: %v", r)
		}
	}()

	counterKey := fmt.Sprintf("af:dynbl:cnt:%s", ip)
	count, err := rdbLocal.Incr(ctxLocal, counterKey).Result()
	if err != nil {
		log.Printf("Error incrementing dynamic blacklist counter: %v", err)
		return
	}
	if count == 1 {
		rdbLocal.Expire(ctxLocal, counterKey, dynBlacklistWindow)
	}
	if count >= int64(threshold) {
		promoteToDynamicBlacklist(ip, rdbLocal)
		rdbLocal.Del(ctxLocal, counterKey)
	}
}
