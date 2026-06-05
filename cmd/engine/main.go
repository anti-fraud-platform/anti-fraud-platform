package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type ClickRequest struct {
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
	CampaignID string `json:"campaign_id"`
}

type ClickResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/click", handleClick)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Println("The Core Engine is running on the port :8080...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}

func handleClick(w http.ResponseWriter, r *http.Request) {
	var req ClickRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[Click Received] IP: %s | UA: %s | Campaign: %s\n", req.IP, req.UserAgent, req.CampaignID)

	response := ClickResponse{
		Allowed: true,
		Reason:  "passed_initial_check",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)
}
