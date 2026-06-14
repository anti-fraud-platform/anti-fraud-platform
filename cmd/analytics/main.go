package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type StatsResponse struct {
	TotalClicks   int64   `json:"total_clicks"`
	BlockedBots   int64   `json:"blocked_bots"`
	SavedMoneyUSD float64 `json:"saved_money_usd"`
}

var db *sql.DB

func statsHandler(w http.ResponseWriter, r *http.Request) {
	var totalClicks, blockedBots int64

	err := db.QueryRow("SELECT COUNT(*) FROM click_logs").Scan(&totalClicks)
	if err != nil {
		log.Printf("Error counting total clicks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE is_bot = true").Scan(&blockedBots)
	if err != nil {
		log.Printf("Error counting blocked bots: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	saved := float64(blockedBots) * 5.0

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(StatsResponse{
		TotalClicks:   totalClicks,
		BlockedBots:   blockedBots,
		SavedMoneyUSD: saved,
	})
}

func main() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5433"
	}

	connStr := "user=antifraud password=antifraud123 dbname=analytics host=" + dbHost + " port=" + dbPort + " sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("DB not reachable:", err)
	}
	log.Println("Connected to PG")

	http.HandleFunc("/v1/analytics/stats", statsHandler)
	log.Println("Analytics on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
