package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"anti-fraud/internal/auth"

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

func TestUpdateCampaignHandlerRejectsNonPUT(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/analytics/campaigns", nil)
	w := httptest.NewRecorder()

	updateCampaignHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestUpdateCampaignHandlerRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/v1/analytics/campaigns", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	updateCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateCampaignHandlerRejectsMissingCampaignID(t *testing.T) {
	body := []byte(`{"cost_per_click":10}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/analytics/campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	updateCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateCampaignHandlerRejectsNonPositiveCostPerClick(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{
			name: "zero",
			body: `{"campaign_id":"demo","cost_per_click":0}`,
		},
		{
			name: "negative",
			body: `{"campaign_id":"demo","cost_per_click":-5}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/v1/analytics/campaigns", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			updateCampaignHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestUpdateCampaignHandlerUpsertsCampaignCost(t *testing.T) {
	setupTestDB(t)

	testCampaignID := "test_update_campaign_cost"
	defer db.Exec("DELETE FROM campaigns WHERE campaign_id = $1", testCampaignID)

	for _, cost := range []int64{17, 23} {
		body := []byte(fmt.Sprintf(`{"campaign_id":"%s","cost_per_click":%d}`, testCampaignID, cost))
		req := httptest.NewRequest(http.MethodPut, "/v1/analytics/campaigns", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		updateCampaignHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if got, _ := resp["campaign_id"].(string); got != testCampaignID {
			t.Fatalf("expected campaign_id %q, got %#v", testCampaignID, resp["campaign_id"])
		}
		if got, ok := resp["cost_per_click"].(float64); !ok || int64(got) != cost {
			t.Fatalf("expected cost_per_click %d, got %#v", cost, resp["cost_per_click"])
		}

		var storedCost int64
		if err := db.QueryRow("SELECT cost_per_click FROM campaigns WHERE campaign_id = $1", testCampaignID).Scan(&storedCost); err != nil {
			t.Fatalf("failed to query stored campaign cost: %v", err)
		}
		if storedCost != cost {
			t.Fatalf("expected stored cost %d, got %d", cost, storedCost)
		}
	}
}

func TestAnalyticsProtectedFlowWithJWT(t *testing.T) {
	setupTestDB(t)

	store := auth.NewUserStore(db)
	authHandlers := auth.NewAuthHandlers(store)
	authHandlers.SeedAdmin("admin", "admin123")

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", authHandlers.LoginHandler)
	mux.Handle("/v1/auth/me", auth.RequireAuth(http.HandlerFunc(authHandlers.MeHandler)))
	mux.Handle("/v1/analytics/stats", auth.RequireAuth(http.HandlerFunc(statsHandler)))

	unauthReq := httptest.NewRequest(http.MethodGet, "/v1/analytics/stats", nil)
	unauthResp := httptest.NewRecorder()
	mux.ServeHTTP(unauthResp, unauthReq)

	if unauthResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", unauthResp.Code)
	}

	loginBody := []byte(`{"username":"admin","password":"admin123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	mux.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from login, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	if tokenResp.Token == "" {
		t.Fatal("expected non-empty token from login")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
	meResp := httptest.NewRecorder()
	mux.ServeHTTP(meResp, meReq)

	if meResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from /v1/auth/me, got %d: %s", meResp.Code, meResp.Body.String())
	}

	var mePayload struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := json.Unmarshal(meResp.Body.Bytes(), &mePayload); err != nil {
		t.Fatalf("failed to parse /v1/auth/me response: %v", err)
	}
	if mePayload.Username != "admin" {
		t.Fatalf("expected username admin, got %q", mePayload.Username)
	}
	if mePayload.Role != "admin" {
		t.Fatalf("expected role admin, got %q", mePayload.Role)
	}

	statsReq := httptest.NewRequest(http.MethodGet, "/v1/analytics/stats", nil)
	statsReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
	statsResp := httptest.NewRecorder()
	mux.ServeHTTP(statsResp, statsReq)

	if statsResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from authorized /v1/analytics/stats, got %d: %s", statsResp.Code, statsResp.Body.String())
	}
}
