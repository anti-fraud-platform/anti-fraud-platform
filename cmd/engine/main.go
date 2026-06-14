package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type MemoryRateLimiter struct {
	mu       sync.Mutex
	requests map[string]int
	maxRate  int
}

type ClickPayload struct {
	CampaignID string `json:"campaign_id"`
	UserAgent  string `json:"user_agent"`
}

var limiter = &MemoryRateLimiter{
	requests: make(map[string]int),
	maxRate:  5,
}

func main() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			limiter.mu.Lock()
			limiter.requests = make(map[string]int)
			limiter.mu.Unlock()
		}
	}()

	http.HandleFunc("/v1/click", rateLimitMiddleware(handleClick))

	log.Println("Core Engine API Gateway started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		limiter.mu.Lock()
		limiter.requests[ip]++
		currentRequests := limiter.requests[ip]
		limiter.mu.Unlock()

		if currentRequests > limiter.maxRate {
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
