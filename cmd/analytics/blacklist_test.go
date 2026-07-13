package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) {
	if db != nil {
		return
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "antifraud"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "antifraud123"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "analytics"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test: database not available (%v)", err)
		return
	}
	if err = db.Ping(); err != nil {
		t.Skipf("Skipping test: database not reachable (%v)", err)
		return
	}
}

func TestBlacklistEndpointNotEmpty(t *testing.T) {
	setupTestDB(t)

	testIP := "192.168.1.1_regression_test"

	_, err := db.Exec("INSERT INTO dynamic_blacklist (ip, reason) VALUES ($1, 'test') ON CONFLICT (ip) DO NOTHING", testIP)
	if err != nil {
		t.Fatalf("Failed to insert test IP: %v", err)
	}
	defer db.Exec("DELETE FROM dynamic_blacklist WHERE ip = $1", testIP)

	req := httptest.NewRequest("GET", "/v1/analytics/blacklist/ips", nil)
	w := httptest.NewRecorder()

	blacklistIPsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp BlacklistIPsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if resp.Total == 0 || len(resp.Items) == 0 {
		t.Errorf("Regression failed: expected blacklist response to not be empty, but got total=%d, items=%d", resp.Total, len(resp.Items))
	}
}

func TestStatsHandlerCostPerClickCalculation(t *testing.T) {
	setupTestDB(t)

	testCampaignID := "test_cpc_campaign_999"
	testCPC := 15.50
	testBlockedClicks := 4

	_, err := db.Exec(`
		INSERT INTO campaigns (campaign_id, cost_per_click) 
		VALUES ($1, $2) 
		ON CONFLICT (campaign_id) DO UPDATE SET cost_per_click = $2`,
		testCampaignID, testCPC)
	if err != nil {
		t.Skipf("Skipping test: campaigns table might not exist yet (%v)", err)
	}
	defer db.Exec("DELETE FROM campaigns WHERE campaign_id = $1", testCampaignID)

	for i := 0; i < testBlockedClicks; i++ {
		_, err := db.Exec(`
			INSERT INTO click_logs (ip, campaign_id, is_bot, reason, processed_at) 
			VALUES ($1, $2, true, 'test_cpc_check', NOW())`,
			fmt.Sprintf("10.0.0.%d", i), testCampaignID)
		if err != nil {
			t.Fatalf("Failed to insert test click log: %v", err)
		}
	}
	defer db.Exec("DELETE FROM click_logs WHERE campaign_id = $1", testCampaignID)

	req := httptest.NewRequest("GET", "/v1/analytics/stats", nil)
	w := httptest.NewRecorder()
	statsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse stats response JSON: %v", err)
	}

	var found bool
	expectedSavedMoney := float64(testBlockedClicks) * testCPC // 4 * 15.50 = 62.00

	for _, camp := range resp.Campaigns {
		if camp.CampaignID == testCampaignID {
			found = true
			if camp.SavedMoneyUSD < expectedSavedMoney-0.01 || camp.SavedMoneyUSD > expectedSavedMoney+0.01 {
				t.Errorf("Expected SavedMoneyUSD to be ~%.2f (based on real CPC), but got %.2f", expectedSavedMoney, camp.SavedMoneyUSD)
			}
			break
		}
	}

	if !found {
		t.Errorf("Test campaign %s not found in stats response. Campaigns returned: %v", testCampaignID, resp.Campaigns)
	}
}
