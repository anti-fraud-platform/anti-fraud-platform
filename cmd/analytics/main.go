package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// ---------- Response Structures ----------

// CampaignStats holds aggregated data per campaign.
type CampaignStats struct {
	CampaignID    string  `json:"campaign_id"`
	TotalClicks   int64   `json:"total_clicks"`
	BlockedBots   int64   `json:"blocked_bots"`
	SavedMoneyUSD float64 `json:"saved_money_usd"`
}

// StatsResponse is the full JSON response for /v1/analytics/stats.
type StatsResponse struct {
	TotalClicks   int64           `json:"total_clicks"`
	BlockedBots   int64           `json:"blocked_bots"`
	SavedMoneyUSD float64         `json:"saved_money_usd"`
	Campaigns     []CampaignStats `json:"campaigns"`
}

// ClickLogEntry represents a single row from the click_logs table.
type ClickLogEntry struct {
	ID          int64     `json:"id"`
	IP          string    `json:"ip"`
	CampaignID  string    `json:"campaign_id"`
	UserAgent   string    `json:"user_agent"`
	IsBot       bool      `json:"is_bot"`
	Reason      string    `json:"reason"`
	ProcessedAt time.Time `json:"processed_at"`
}

// LogsResponse is the paginated response for /v1/analytics/logs.
type LogsResponse struct {
	Data       []ClickLogEntry `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

// ---------- Global Variables ----------

var db *sql.DB

const maxLimit = 100 // maximum page size to prevent abuse

// ---------- Handlers ----------

// statsHandler returns overall statistics and per-campaign aggregation.
func statsHandler(w http.ResponseWriter, r *http.Request) {
	// Overall counts
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

	// Group by campaign
	rows, err := db.Query(`
		SELECT campaign_id, COUNT(*) as total, COUNT(*) FILTER (WHERE is_bot = true) as blocked
		FROM click_logs
		GROUP BY campaign_id
		ORDER BY campaign_id
	`)
	if err != nil {
		log.Printf("Error querying campaign stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	campaigns := []CampaignStats{}
	for rows.Next() {
		var campID string
		var total, blocked int64
		if err := rows.Scan(&campID, &total, &blocked); err != nil {
			log.Printf("Error scanning campaign stats: %v", err)
			continue
		}
		campaigns = append(campaigns, CampaignStats{
			CampaignID:    campID,
			TotalClicks:   total,
			BlockedBots:   blocked,
			SavedMoneyUSD: float64(blocked) * 5.0, // $5 saved per blocked bot
		})
	}

	saved := float64(blockedBots) * 5.0

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(StatsResponse{
		TotalClicks:   totalClicks,
		BlockedBots:   blockedBots,
		SavedMoneyUSD: saved,
		Campaigns:     campaigns,
	})
}

// logsHandler returns raw click logs with pagination and filtering.
func logsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")
	campaignID := query.Get("campaign_id")
	isBotStr := query.Get("is_bot")
	reason := query.Get("reason")

	// Parse page number (default 1)
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p >= 1 {
		page = p
	}

	// Parse limit with upper bound
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil {
		if l >= 1 && l <= maxLimit {
			limit = l
		} else if l > maxLimit {
			limit = maxLimit
		}
	}

	// Parse is_bot as optional boolean
	var isBot *bool
	if isBotStr != "" {
		b, err := strconv.ParseBool(isBotStr)
		if err == nil {
			isBot = &b
		}
	}

	// Build WHERE clause dynamically
	whereParts := []string{}
	args := []interface{}{}
	argCounter := 1

	if campaignID != "" {
		whereParts = append(whereParts, fmt.Sprintf("campaign_id = $%d", argCounter))
		args = append(args, campaignID)
		argCounter++
	}
	if isBot != nil {
		whereParts = append(whereParts, fmt.Sprintf("is_bot = $%d", argCounter))
		args = append(args, *isBot)
		argCounter++
	}
	if reason != "" {
		whereParts = append(whereParts, fmt.Sprintf("reason = $%d", argCounter))
		args = append(args, reason)
		argCounter++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	// Count total matching records (for pagination metadata)
	countQuery := "SELECT COUNT(*) FROM click_logs " + whereClause
	var total int64
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("Error counting logs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Fetch actual data with ordering and limit/offset
	dataQuery := fmt.Sprintf(`
		SELECT id, ip, campaign_id, user_agent, is_bot, reason, processed_at
		FROM click_logs
		%s
		ORDER BY processed_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCounter, argCounter+1)

	dataArgs := append(args, limit, offset)
	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		log.Printf("Error querying logs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []ClickLogEntry{}
	for rows.Next() {
		var entry ClickLogEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.IP,
			&entry.CampaignID,
			&entry.UserAgent,
			&entry.IsBot,
			&entry.Reason,
			&entry.ProcessedAt,
		); err != nil {
			log.Printf("Error scanning log row: %v", err)
			continue
		}
		logs = append(logs, entry)
	}

	response := LogsResponse{
		Data:       logs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

// ---------- Helper Functions ----------

// getEnv retrieves an environment variable or returns a fallback value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// ---------- Main Entry Point ----------

func main() {
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
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("DB not reachable:", err)
	}
	log.Println("Connected to PostgreSQL")

	http.HandleFunc("/v1/analytics/stats", statsHandler)
	http.HandleFunc("/v1/analytics/logs", logsHandler)

	port := getEnv("ANALYTICS_PORT", "8081")
	log.Printf("Analytics service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}