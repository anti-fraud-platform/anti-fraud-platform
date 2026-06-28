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
	"anti-fraud/internal/logger"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type ClickPayload struct {
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
	CampaignID string `json:"campaign_id"`
	Timestamp  int64  `json:"timestamp"`
}

var (
	rdb         *redis.Client
	db          *sql.DB
	ipFilter    *bloom.IPFilter
	batchLogger *logger.BatchLogger
	ctx         = context.Background()
	maxRate     = 5
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

	http.HandleFunc("/v1/click", handleClick)

	log.Println("Core Engine API Gateway started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
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

	key := fmt.Sprintf("rate:%s", ip)
	currentRequests, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("Redis error: %v", err)
	} else {
		// ExpireNX only sets a TTL if the key doesn't already have one.
		// Calling this on every request (not just when currentRequests
		// == 1) means a single dropped/failed Expire call can never
		// permanently strand a key without a TTL — every subsequent
		// request gets a chance to set it instead. Without this, a key
		// that loses its TTL once will INCR forever and the IP gets
		// rate-limited permanently, even after the attack stops.
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
