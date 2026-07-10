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
		t.Fatalf("Failed to open test DB connection: %v", err)
	}
	if err = db.Ping(); err != nil {
		t.Fatalf("Test DB not reachable: %v", err)
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
