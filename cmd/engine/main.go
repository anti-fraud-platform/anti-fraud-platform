package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type ClickPayload struct {
	CampaignID string `json:"campaign_id"`
	UserAgent  string `json:"user_agent"`
}

var (
	rdb     *redis.Client
	ctx     = context.Background()
	maxRate = 5
)

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	if redisHost == "" {
		redisHost = "localhost"
	}
	if redisPort == "" {
		redisPort = "6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis storage")

	http.HandleFunc("/v1/click", rateLimitMiddleware(handleClick))

	log.Println("Core Engine API Gateway started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests) // HTTP 429
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
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Click registered, routing to verification queue",
	})
}
