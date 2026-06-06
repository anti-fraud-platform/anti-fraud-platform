package main

import (
    "encoding/json"
    "log"
    "net/http"
)

type StatsResponse struct {
    TotalClicks   int64   `json:"total_clicks"`
    BlockedBots   int64   `json:"blocked_bots"`
    SavedMoneyUSD float64 `json:"saved_money_usd"`
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
    resp := StatsResponse{
        TotalClicks:   12500,
        BlockedBots:   4980,
        SavedMoneyUSD: 24900.00,
    }
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    json.NewEncoder(w).Encode(resp)
}

func main() {
    http.HandleFunc("/v1/analytics/stats", statsHandler)
    log.Println("Analytics server on :8081")
    log.Fatal(http.ListenAndServe(":8081", nil))
}