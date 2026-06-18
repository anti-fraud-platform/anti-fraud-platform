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
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"anti-fraud/internal/bloom"
	"anti-fraud/internal/logger"
)

type ClickPayload struct {
	CampaignID string `json:"campaign_id"`
	UserAgent  string `json:"user_agent"`
}

var (
	rdb         *redis.Client
	ctx         = context.Background()
	maxRate     = 5
	blacklist   *bloom.IPFilter
	batchLogger *logger.BatchLogger
)

func main() {
	// --- Redis (rate limiter state) ---
	redisHost := getenv("REDIS_HOST", "localhost")
	redisPort := getenv("REDIS_PORT", "6379")

	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis storage")

	// --- Bloom filter (known-bad IP blacklist) ---
	blacklistPath := getenv("BLACKLIST_PATH", "deployments/blacklists/dirty_ips.txt")
	var err error
	blacklist, err = bloom.NewIPFilter(blacklistPath)
	if err != nil {
		log.Fatalf("Failed to load blacklist: %v", err)
	}

	// --- Postgres (async click logging) ---
	pgConnStr := getenv("POSTGRES_CONN", "host=localhost port=5432 user=postgres password=postgres dbname=antifraud sslmode=disable")
	db, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("Failed to open Postgres connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	log.Println("Successfully connected to Postgres")

	// batchSize=200, flushInterval=2000ms
	batchLogger = logger.NewBatchLogger(db, 200, 2000)
	batchLogger.Start(ctx)

	// --- HTTP server ---
	// Order matters: Bloom filter (nanoseconds, no network) runs
	// before the Redis rate limiter (milliseconds, network call).
	// Known-bad IPs get rejected before ever touching Redis.
	http.HandleFunc("/v1/click", blacklistMiddleware(rateLimitMiddleware(handleClick)))

	log.Println("Core Engine API Gateway started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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

// blacklistMiddleware is the FIRST line of defense. Bloom filter
// lookup costs nanoseconds and needs no network call, so it runs
// before the Redis rate-limit check. Known-bad IPs get rejected
// before ever touching Redis.
func blacklistMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if blacklist.IsBlacklisted(ip) {
			log.Printf("[BLACKLIST] Blocked known-bad IP: %s", ip)

			payload := decodePayloadBestEffort(r)
			batchLogger.LogAsync(logger.ClickLog{
				IP:         ip,
				CampaignID: payload.CampaignID,
				UserAgent:  payload.UserAgent,
				IsBot:      true,
				Reason:     "blacklisted_ip",
			})

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden) // 403
			json.NewEncoder(w).Encode(map[string]string{"error": "IP is blacklisted"})
			return
		}

		next.ServeHTTP(w, r)
	}
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		key := fmt.Sprintf("rate:%s", ip)

		currentRequests, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("Redis error: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		if currentRequests == 1 {
			rdb.Expire(ctx, key, time.Second)
		}

		if int(currentRequests) > maxRate {
			log.Printf("[RATE LIMIT] Blocked malicious requests from IP: %s (Rate: %d/s)", ip, currentRequests)

			payload := decodePayloadBestEffort(r)
			batchLogger.LogAsync(logger.ClickLog{
				IP:         ip,
				CampaignID: payload.CampaignID,
				UserAgent:  payload.UserAgent,
				IsBot:      true,
				Reason:     "rate_limited",
			})

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests) // 429
			json.NewEncoder(w).Encode(map[string]string{"error": "Too many requests. Real-time anti-fraud trigger."})
			return
		}

		next.ServeHTTP(w, r)
	}
}

func handleClick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload ClickPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ip := getClientIP(r)
	batchLogger.LogAsync(logger.ClickLog{
		IP:         ip,
		CampaignID: payload.CampaignID,
		UserAgent:  payload.UserAgent,
		IsBot:      false,
		Reason:     "allowed",
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Click registered, routing to verification queue",
	})
}

// decodePayloadBestEffort reads the body for logging on rejected
// requests. Safe because we return immediately after — no later
// handler will try to read the already-drained body.
func decodePayloadBestEffort(r *http.Request) ClickPayload {
	var payload ClickPayload
	_ = json.NewDecoder(r.Body).Decode(&payload)
	return payload
}