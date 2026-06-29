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

// ---------- Constants ----------

// costPerClickUSD is the fixed CPC estimate used to compute the ad budget
// saved by blocking a fraudulent click. Change this single value to adjust
// every "budget saved" figure the service reports.
const costPerClickUSD = 5.0

// topBlockedIPsLimit caps how many offending IPs the stats endpoint returns.
const topBlockedIPsLimit = 10

// ---------- Response Structures ----------

// CampaignStats holds aggregated data per campaign.
type CampaignStats struct {
	CampaignID    string  `json:"campaign_id"`
	TotalClicks   int64   `json:"total_clicks"`
	BlockedBots   int64   `json:"blocked_bots"`
	SavedMoneyUSD float64 `json:"saved_money_usd"`
}

// BlockedIPStat holds how many times a single IP was blocked.
type BlockedIPStat struct {
	IP    string `json:"ip"`
	Count int64  `json:"count"`
}

// StatsResponse is the full JSON response for /v1/analytics/stats.
type StatsResponse struct {
	TotalClicks   int64           `json:"total_clicks"`
	AllowedCount  int64           `json:"allowed_count"`
	BlockedCount  int64           `json:"blocked_count"`
	BlockedBots   int64           `json:"blocked_bots"` // kept for backward compatibility with the existing frontend
	SavedMoneyUSD float64         `json:"saved_money_usd"`
	BudgetSaved   float64         `json:"budget_saved"` // blocked_count * costPerClickUSD
	TopBlockedIPs []BlockedIPStat `json:"top_blocked_ips"`
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

// statsHandler returns overall statistics, per-campaign aggregation,
// and the top offending IPs. This is what the frontend reads on initial
// page load, before the WebSocket stream takes over for live updates.
func statsHandler(w http.ResponseWriter, r *http.Request) {
	// Overall counts: total clicks and how many were blocked as bots.
	var totalClicks, blockedCount int64
	if err := db.QueryRow("SELECT COUNT(*) FROM click_logs").Scan(&totalClicks); err != nil {
		log.Printf("Error counting total clicks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE is_bot = true").Scan(&blockedCount); err != nil {
		log.Printf("Error counting blocked clicks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Allowed = everything that was not blocked.
	allowedCount := totalClicks - blockedCount

	// Per-campaign aggregation.
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
			SavedMoneyUSD: float64(blocked) * costPerClickUSD,
		})
	}

	// Top blocked IPs: which addresses were flagged as bots most often.
	topRows, err := db.Query(`
		SELECT ip, COUNT(*) as cnt
		FROM click_logs
		WHERE is_bot = true
		GROUP BY ip
		ORDER BY cnt DESC
		LIMIT $1
	`, topBlockedIPsLimit)
	if err != nil {
		log.Printf("Error querying top blocked IPs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer topRows.Close()

	topBlockedIPs := []BlockedIPStat{}
	for topRows.Next() {
		var stat BlockedIPStat
		if err := topRows.Scan(&stat.IP, &stat.Count); err != nil {
			log.Printf("Error scanning top blocked IP: %v", err)
			continue
		}
		topBlockedIPs = append(topBlockedIPs, stat)
	}

	saved := float64(blockedCount) * costPerClickUSD

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(StatsResponse{
		TotalClicks:   totalClicks,
		AllowedCount:  allowedCount,
		BlockedCount:  blockedCount,
		BlockedBots:   blockedCount, // same value, kept so existing frontend keeps working
		SavedMoneyUSD: saved,
		BudgetSaved:   saved,
		TopBlockedIPs: topBlockedIPs,
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
	fromStr := query.Get("from")
	toStr := query.Get("to")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p >= 1 {
		page = p
	}

	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil {
		if l >= 1 && l <= maxLimit {
			limit = l
		} else if l > maxLimit {
			limit = maxLimit
		}
	}

	var isBot *bool
	if isBotStr != "" {
		b, err := strconv.ParseBool(isBotStr)
		if err == nil {
			isBot = &b
		}
	}

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

	if fromStr != "" {
		tFrom, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			tFrom, err = time.Parse("2006-01-02", fromStr)
		}
		if err == nil {
			whereParts = append(whereParts, fmt.Sprintf("processed_at >= $%d", argCounter))
			args = append(args, tFrom)
			argCounter++
		}
	}

	if toStr != "" {
		tTo, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			tTo, err = time.Parse("2006-01-02", toStr)
		}
		if err == nil {
			if !strings.Contains(toStr, "T") {
				tTo = tTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			}
			whereParts = append(whereParts, fmt.Sprintf("processed_at <= $%d", argCounter))
			args = append(args, tTo)
			argCounter++
		}
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM click_logs " + whereClause
	var total int64
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("Error counting logs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

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
