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

	"anti-fraud/internal/dbschema"

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
	IP            string `json:"ip"`
	Count         int64  `json:"blocked"`        // backward compatible
	TotalRequests int64  `json:"total_requests"` // new
}

// StatsResponse is the full JSON response for /v1/analytics/stats.
type StatsResponse struct {
	TotalClicks          int64           `json:"total_clicks"`
	AllowedCount         int64           `json:"allowed_count"`
	BlockedCount         int64           `json:"blocked_count"`
	BlockedBots          int64           `json:"blocked_bots"` // kept for backward compatibility with the existing frontend
	SavedMoneyUSD        float64         `json:"saved_money_usd"`
	BudgetSaved          float64         `json:"budget_saved"` // blocked_count * costPerClickUSD
	TopBlockedIPs        []BlockedIPStat `json:"top_blocked_ips"`
	Campaigns            []CampaignStats `json:"campaigns"`
	PreviousTotalClicks  int64           `json:"previous_total_clicks"`
	PreviousBlockedCount int64           `json:"previous_blocked_count"`
	TotalClicksDelta     float64         `json:"total_clicks_delta_percent"`
	BlockedCountDelta    float64         `json:"blocked_count_delta_percent"`

	// ReasonBreakdown maps every distinct click_logs.reason value (blocked
	// ones only) to its count, e.g. {"suspicious_agent": 12,
	// "no_js_challenge": 340, "suspicious_headers": 88,
	// "static_blacklist": 4, "rate_limit_exceeded": 900}. Lets the
	// dashboard show every detection layer's contribution without the
	// backend needing to add a new named field each time a new check ships.
	ReasonBreakdown map[string]int64 `json:"reason_breakdown"`

	// Flat convenience fields for the two Tier-1 checks specifically,
	// mirroring the existing blocked_bots-style convenience pattern —
	// handy for a single stat card without reading the map.
	JSChallengeBlocked     int64 `json:"js_challenge_blocked"`
	HeaderHeuristicBlocked int64 `json:"header_heuristic_blocked"`
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

// BlacklistSummaryResponse holds statistics for the blacklist dashboard metrics.
type BlacklistSummaryResponse struct {
	TotalBlocked    int64 `json:"total_blocked"`
	StaticBlacklist int64 `json:"static_blacklist"`
	RateLimited     int64 `json:"rate_limited"`
	AutoBlocked24h  int64 `json:"auto_blocked_24h"`

	// New Tier-1 detection layers, broken out the same way the existing
	// fields are, so the Blacklist page's summary cards can show them
	// without a schema change (reason is already TEXT).
	JSChallengeBlocked     int64 `json:"js_challenge_blocked"`
	HeaderHeuristicBlocked int64 `json:"header_heuristic_blocked"`
}

// DailyTrend holds aggregated traffic data for a single day.
type DailyTrend struct {
	Date         string `json:"date"` // Формат: YYYY-MM-DD
	TotalClicks  int64  `json:"total_clicks"`
	AllowedCount int64  `json:"allowed_count"`
	BlockedCount int64  `json:"blocked_count"`
}

// TrendResponse represents the array payload for the 7-day chart.
type TrendResponse struct {
	Data []DailyTrend `json:"data"`
}

// AuditEvent represents a system action log entry for the dashboard feed.
type AuditEvent struct {
	ID         int64     `json:"id"`
	ActionText string    `json:"action_text"`
	CreatedAt  time.Time `json:"created_at"`
}

// BlacklistIPEntry represents a single blocked IP with its statistics.
type BlacklistIPEntry struct {
	IP           string `json:"ip"`
	BlockCount   int64  `json:"block_count"`
	FirstBlocked string `json:"first_blocked"`
	LastBlocked  string `json:"last_blocked"`
}

// BlacklistIPsResponse holds the list of blocked IPs.
type BlacklistIPsResponse struct {
	Items []BlacklistIPEntry `json:"items"`
	Total int64              `json:"total"`
}

// ---------- Global Variables ----------

var db *sql.DB

const maxLimit = 100 // maximum page size to prevent abuse

// jsChallengeReasons are the click_logs.reason values produced by the
// challenge package's three failure modes. Grouped together for the
// dashboard because they're all "this client never proved it ran JS",
// even though the specific cause differs.
var jsChallengeReasons = []string{"no_js_challenge", "challenge_too_fast", "challenge_mismatch"}

// ---------- Handlers ----------

// statsHandler returns overall statistics, per-campaign aggregation,
// and the top offending IPs. This is what the frontend reads on initial
// page load, before the WebSocket stream takes over for live updates.
func statsHandler(w http.ResponseWriter, r *http.Request) {
	// Current totals
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
	allowedCount := totalClicks - blockedCount

	// Previous 7‑day period (14 to 7 days ago)
	var prevTotal, prevBlocked int64
	if err := db.QueryRow(`
        SELECT COUNT(*), COUNT(*) FILTER (WHERE is_bot = true)
        FROM click_logs
        WHERE processed_at >= NOW() - INTERVAL '14 days'
          AND processed_at < NOW() - INTERVAL '7 days'
    `).Scan(&prevTotal, &prevBlocked); err != nil {
		log.Printf("Error counting previous period: %v", err)
		prevTotal, prevBlocked = 0, 0
	}

	// Deltas
	deltaTotal := 0.0
	if prevTotal > 0 {
		deltaTotal = float64(totalClicks-prevTotal) / float64(prevTotal) * 100
	}
	deltaBlocked := 0.0
	if prevBlocked > 0 {
		deltaBlocked = float64(blockedCount-prevBlocked) / float64(prevBlocked) * 100
	}

	// Per‑campaign stats with custom cost per click (from campaigns table, default 5.00)
	rows, err := db.Query(`
        SELECT 
            c.campaign_id,
            COALESCE(cam.cost_per_click, 5.00) as cpc,
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE is_bot = true) as blocked
        FROM click_logs c
        LEFT JOIN campaigns cam ON c.campaign_id = cam.campaign_id
        GROUP BY c.campaign_id, cam.cost_per_click
        ORDER BY c.campaign_id
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
		var cpc, total, blocked int64
		if err := rows.Scan(&campID, &cpc, &total, &blocked); err != nil {
			log.Printf("Error scanning campaign stats: %v", err)
			continue
		}
		campaigns = append(campaigns, CampaignStats{
			CampaignID:    campID,
			TotalClicks:   total,
			BlockedBots:   blocked,
			SavedMoneyUSD: float64(blocked) * float64(cpc),
		})
	}

	// Top blocked IPs – now with total requests
	topRows, err := db.Query(`
        SELECT ip,
               COUNT(*) as total_requests,
               COUNT(*) FILTER (WHERE is_bot = true) as blocked_count
        FROM click_logs
        GROUP BY ip
        HAVING COUNT(*) FILTER (WHERE is_bot = true) > 0
        ORDER BY blocked_count DESC
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
		if err := topRows.Scan(&stat.IP, &stat.TotalRequests, &stat.Count); err != nil {
			log.Printf("Error scanning top blocked IP: %v", err)
			continue
		}
		topBlockedIPs = append(topBlockedIPs, stat)
	}

	// Reason breakdown (unchanged)
	reasonRows, err := db.Query(`
        SELECT reason, COUNT(*) as cnt
        FROM click_logs
        WHERE is_bot = true
        GROUP BY reason
    `)
	if err != nil {
		log.Printf("Error querying reason breakdown: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer reasonRows.Close()

	reasonBreakdown := map[string]int64{}
	for reasonRows.Next() {
		var reason string
		var cnt int64
		if err := reasonRows.Scan(&reason, &cnt); err != nil {
			log.Printf("Error scanning reason breakdown row: %v", err)
			continue
		}
		reasonBreakdown[reason] = cnt
	}

	// Convenience fields for JS and header (unchanged)
	var jsChallengeBlocked, headerHeuristicBlocked int64
	jsChallengeReasons := []string{"no_js_challenge", "challenge_too_fast", "challenge_mismatch"}
	for _, reason := range jsChallengeReasons {
		jsChallengeBlocked += reasonBreakdown[reason]
	}
	headerHeuristicBlocked = reasonBreakdown["suspicious_headers"]

	saved := float64(blockedCount) * 5.0 // fallback, will be overridden by per‑campaign sum? We keep this for compatibility.

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(StatsResponse{
		TotalClicks:            totalClicks,
		AllowedCount:           allowedCount,
		BlockedCount:           blockedCount,
		BlockedBots:            blockedCount,
		SavedMoneyUSD:          saved,
		BudgetSaved:            saved,
		TopBlockedIPs:          topBlockedIPs,
		Campaigns:              campaigns,
		ReasonBreakdown:        reasonBreakdown,
		JSChallengeBlocked:     jsChallengeBlocked,
		HeaderHeuristicBlocked: headerHeuristicBlocked,
		// New fields:
		PreviousTotalClicks:  prevTotal,
		PreviousBlockedCount: prevBlocked,
		TotalClicksDelta:     deltaTotal,
		BlockedCountDelta:    deltaBlocked,
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

// blacklistSummaryHandler returns summary metrics specifically for the blacklist page.
func blacklistSummaryHandler(w http.ResponseWriter, r *http.Request) {
	var summary BlacklistSummaryResponse

	err := db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE is_bot = true").Scan(&summary.TotalBlocked)
	if err != nil {
		log.Printf("Error querying total blocked: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE reason = 'static_blacklist'").Scan(&summary.StaticBlacklist)
	if err != nil {
		log.Printf("Error querying static blacklist count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE reason = 'rate_limit_exceeded'").Scan(&summary.RateLimited)
	if err != nil {
		log.Printf("Error querying rate limit count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE is_bot = true AND processed_at >= NOW() - INTERVAL '24 hours'").Scan(&summary.AutoBlocked24h)
	if err != nil {
		log.Printf("Error querying 24h blocked count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE reason IN ('no_js_challenge', 'challenge_too_fast', 'challenge_mismatch')").Scan(&summary.JSChallengeBlocked)
	if err != nil {
		log.Printf("Error querying js challenge blocked count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow("SELECT COUNT(*) FROM click_logs WHERE reason = 'suspicious_headers'").Scan(&summary.HeaderHeuristicBlocked)
	if err != nil {
		log.Printf("Error querying header heuristic blocked count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(summary)
}

// trendHandler returns day-over-day aggregated traffic for the last 7 days.
func trendHandler(w http.ResponseWriter, r *http.Request) {
	type DailyTrend struct {
		Date         string `json:"date"`
		TotalClicks  int64  `json:"total_clicks"`
		AllowedCount int64  `json:"allowed_count"`
		BlockedCount int64  `json:"blocked_count"`
	}
	type DailyTrendWithBreakdown struct {
		DailyTrend
		Breakdown map[string]int64 `json:"breakdown"`
	}

	rows, err := db.Query(`
        SELECT 
            TO_CHAR(processed_at, 'YYYY-MM-DD') as log_date,
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE is_bot = false) as allowed,
            COUNT(*) FILTER (WHERE is_bot = true) as blocked
        FROM click_logs
        WHERE processed_at >= NOW() - INTERVAL '7 days'
        GROUP BY log_date
        ORDER BY log_date ASC
    `)
	if err != nil {
		log.Printf("Error querying trend: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type SimpleTrend struct {
		Date    string
		Total   int64
		Allowed int64
		Blocked int64
	}
	var trends []SimpleTrend
	for rows.Next() {
		var t SimpleTrend
		if err := rows.Scan(&t.Date, &t.Total, &t.Allowed, &t.Blocked); err != nil {
			log.Printf("Error scanning trend row: %v", err)
			continue
		}
		trends = append(trends, t)
	}

	result := make([]DailyTrendWithBreakdown, 0, len(trends))
	for _, t := range trends {
		breakdownRows, err := db.Query(`
            SELECT reason, COUNT(*) as cnt
            FROM click_logs
            WHERE TO_CHAR(processed_at, 'YYYY-MM-DD') = $1 AND is_bot = true
            GROUP BY reason
        `, t.Date)
		if err != nil {
			log.Printf("Error querying breakdown for %s: %v", t.Date, err)
			continue
		}

		breakdown := map[string]int64{}
		for breakdownRows.Next() {
			var reason string
			var cnt int64
			if err := breakdownRows.Scan(&reason, &cnt); err != nil {
				log.Printf("Error scanning breakdown row: %v", err)
				continue
			}
			breakdown[reason] = cnt
		}
		breakdownRows.Close()

		result = append(result, DailyTrendWithBreakdown{
			DailyTrend: DailyTrend{
				Date:         t.Date,
				TotalClicks:  t.Total,
				AllowedCount: t.Allowed,
				BlockedCount: t.Blocked,
			},
			Breakdown: breakdown,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": result})
}

// auditEventsHandler returns the latest system events for the recent activity feed.
func auditEventsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, action_text, created_at FROM audit_events ORDER BY created_at DESC LIMIT 20")
	if err != nil {
		log.Printf("Error querying audit events: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := []AuditEvent{}
	for rows.Next() {
		var ev AuditEvent
		if err := rows.Scan(&ev.ID, &ev.ActionText, &ev.CreatedAt); err != nil {
			log.Printf("Error scanning audit event: %v", err)
			continue
		}
		events = append(events, ev)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(events)
}

// blacklistIPsHandler returns the list of IPs blocked due to static_blacklist
func blacklistIPsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	query := `
		SELECT 
			ip,
			COUNT(*) as block_count,
			MIN(processed_at) as first_blocked,
			MAX(processed_at) as last_blocked
		FROM click_logs
		WHERE reason = 'static_blacklist'
		GROUP BY ip
		ORDER BY last_blocked DESC
		LIMIT 50
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying blacklist IPs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []BlacklistIPEntry{}
	for rows.Next() {
		var entry BlacklistIPEntry
		var firstBlocked, lastBlocked time.Time
		if err := rows.Scan(&entry.IP, &entry.BlockCount, &firstBlocked, &lastBlocked); err != nil {
			log.Printf("Error scanning blacklist row: %v", err)
			continue
		}
		entry.FirstBlocked = firstBlocked.Format("2006-01-02 15:04")
		entry.LastBlocked = lastBlocked.Format("2006-01-02 15:04")
		items = append(items, entry)
	}

	var total int64
	db.QueryRow("SELECT COUNT(DISTINCT ip) FROM click_logs WHERE reason = 'static_blacklist'").Scan(&total)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(BlacklistIPsResponse{
		Items: items,
		Total: total,
	})
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
	http.HandleFunc("/health", healthHandler)

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

	if err = dbschema.Apply(db); err != nil {
		log.Fatal("Schema migration failed:", err)
	}
	log.Println("PostgreSQL schema is up to date")

	http.HandleFunc("/v1/analytics/stats", statsHandler)
	http.HandleFunc("/v1/analytics/logs", logsHandler)
	http.HandleFunc("/v1/analytics/blacklist/summary", blacklistSummaryHandler)
	http.HandleFunc("/v1/analytics/blacklist/ips", blacklistIPsHandler)
	http.HandleFunc("/v1/analytics/trend", trendHandler)
	http.HandleFunc("/v1/analytics/events", auditEventsHandler)

	port := getEnv("ANALYTICS_PORT", "8081")
	log.Printf("Analytics service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// healthHandler returns PostgreSQL health status.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{}
	if err := db.Ping(); err != nil {
		status["postgres"] = "unhealthy"
	} else {
		status["postgres"] = "healthy"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
