package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"anti-fraud/internal/bloom"
	"anti-fraud/internal/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestEngine(t *testing.T) func() {
	t.Helper()

	mr := miniredis.RunT(t)

	rdb = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	ipFilter = nil
	batchLogger = logger.NewBatchLogger(nil, 100, 1000)
	maxRate = 5

	return func() {
		_ = rdb.Close()
		mr.Close()
	}
}

func performClickRequest(method string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/v1/click", bytes.NewReader([]byte(body)))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	handleClick(rr, req)
	return rr
}

func TestHandleClickTableDriven(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:   "valid JSON body returns 200",
			method: http.MethodPost,
			body: `{
				"ip":"9.9.9.9",
				"user_agent":"test-agent",
				"campaign_id":"camp_test",
				"timestamp":123456789
			}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "valid JSON without IP uses request remote address and returns 200",
			method: http.MethodPost,
			body: `{
				"user_agent":"test-agent",
				"campaign_id":"camp_no_ip",
				"timestamp":123456789
			}`,
			headers: map[string]string{
				"Content-Type":    "application/json",
				"X-Forwarded-For": "8.8.8.8",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "invalid JSON still handled with defaults and returns 200",
			method: http.MethodPost,
			body:   `{bad json`,
			headers: map[string]string{
				"Content-Type":    "application/json",
				"X-Forwarded-For": "7.7.7.7",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET is rejected with 405",
			method:     http.MethodGet,
			body:       "",
			headers:    nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT is rejected with 405",
			method:     http.MethodPut,
			body:       "",
			headers:    nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEngine(t)
			defer cleanup()

			rr := performClickRequest(tt.method, tt.body, tt.headers)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleClickRateLimitTableDriven(t *testing.T) {
	tests := []struct {
		name             string
		ip               string
		requestsToSend   int
		wantLastStatus   int
		wantBlockedCount int
	}{
		{
			name:             "same IP is blocked after maxRate",
			ip:               "1.2.3.4",
			requestsToSend:   maxRate + 1,
			wantLastStatus:   http.StatusTooManyRequests,
			wantBlockedCount: 1,
		},
		{
			name:             "same IP heavily spammed gets multiple 429s",
			ip:               "2.2.2.2",
			requestsToSend:   maxRate + 5,
			wantLastStatus:   http.StatusTooManyRequests,
			wantBlockedCount: 5,
		},
		{
			name:             "requests within limit stay allowed",
			ip:               "3.3.3.3",
			requestsToSend:   maxRate,
			wantLastStatus:   http.StatusOK,
			wantBlockedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEngine(t)
			defer cleanup()

			blockedCount := 0
			lastStatus := 0

			body := `{
				"user_agent":"test-agent",
				"campaign_id":"camp_rate_limit",
				"timestamp":123456789
			}`

			for i := 0; i < tt.requestsToSend; i++ {
				rr := performClickRequest(http.MethodPost, body, map[string]string{
					"Content-Type":    "application/json",
					"X-Forwarded-For": tt.ip, // real IP now comes from the header, not the body
				})

				lastStatus = rr.Code

				if rr.Code == http.StatusTooManyRequests {
					blockedCount++
				}
			}

			if lastStatus != tt.wantLastStatus {
				t.Fatalf("expected last status %d, got %d", tt.wantLastStatus, lastStatus)
			}

			if blockedCount != tt.wantBlockedCount {
				t.Fatalf("expected blocked count %d, got %d", tt.wantBlockedCount, blockedCount)
			}
		})
	}
}
func TestHandleClickDifferentIPsDoNotShareRateLimit(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()

	for i := 0; i < maxRate+1; i++ {
		body := `{
			"user_agent":"test-agent",
			"campaign_id":"camp_unique_ips",
			"timestamp":123456789
		}`

		rr := performClickRequest(http.MethodPost, body, map[string]string{
			"Content-Type":    "application/json",
			"X-Forwarded-For": "10.0.0." + strconv.Itoa(i+1),
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("expected different IP request to stay 200, got %d, body: %s", rr.Code, rr.Body.String())
		}
	}
}
func TestClickIntegrationPipeline(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()

	testIP := "123.45.67.89"

	ts := httptest.NewServer(http.HandlerFunc(handleClick))
	defer ts.Close()

	client := &http.Client{}

	payload := []byte(`{"campaign_id": "test_campaign", "user_agent": "Mozilla/5.0"}`)
	req, _ := http.NewRequest("POST", ts.URL+"/v1/click", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", testIP)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for clean request, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	for i := 1; i <= maxRate; i++ {
		req, _ := http.NewRequest("POST", ts.URL+"/v1/click", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", testIP)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request during burst: %v", err)
		}

		if i == maxRate {
			if resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Expected status 429 on request number %d, but got %d", i+1, resp.StatusCode)
			}
		} else {
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for request %d under threshold, got %d", i+1, resp.StatusCode)
			}
		}
		resp.Body.Close()
	}
}

func TestHandleClickBloomFilterBlacklist(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_dirty_ips_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	badIP := "99.99.99.99"
	if _, err := tmpFile.WriteString(badIP + "\n"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	var bloomError error
	ipFilter, bloomError = bloom.NewIPFilter(tmpFile.Name())
	if bloomError != nil {
		t.Fatalf("Failed to initialize bloom filter for test: %v", bloomError)
	}
	defer func() { ipFilter = nil }() 

	batchLogger = nil

	body := `{
		"user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		"campaign_id":"camp_bloom_test",
		"timestamp":123456789
	}`

	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": badIP,
	})

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 Forbidden for blacklisted IP, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleClickIgnoresSpoofedBodyIP(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	blockedCount := 0
	lastStatus := 0

	for i := 0; i < maxRate+1; i++ {
		body := `{
			"ip":"9.9.9.` + strconv.Itoa(i+1) + `",
			"user_agent":"test-agent",
			"campaign_id":"camp_spoof_test",
			"timestamp":123456789
		}`

		rr := performClickRequest(http.MethodPost, body, map[string]string{
			"Content-Type": "application/json",
		})

		lastStatus = rr.Code
		if rr.Code == http.StatusTooManyRequests {
			blockedCount++
		}
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("expected spoofed-IP requests to eventually be rate-limited, got last status %d", lastStatus)
	}
	if blockedCount != 1 {
		t.Fatalf("expected exactly 1 blocked request, got %d", blockedCount)
	}
}

func TestHandleClickSelfHealsKeyMissingTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	ipFilter = nil
	batchLogger = logger.NewBatchLogger(nil, 100, 1000)
	maxRate = 5

	ip := "5.5.5.5"
	key := "rate:" + ip

	mr.Set(key, "999")
	if mr.TTL(key) != 0 {
		t.Fatalf("test setup invalid: expected no TTL, got %v", mr.TTL(key))
	}

	body := `{
		"user_agent":"test-agent",
		"campaign_id":"camp_self_heal",
		"timestamp":123456789
	}`

	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": ip,
	})
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected still-blocked on first request, got %d", rr.Code)
	}
	if mr.TTL(key) <= 0 {
		t.Fatalf("expected ExpireNX to have attached a TTL to the previously-stuck key, got %v", mr.TTL(key))
	}
	mr.FastForward(2 * time.Second)

	if mr.Exists(key) {
		t.Fatalf("expected key to have expired after TTL elapsed, but it still exists")
	}

	rr = performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": ip,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to be allowed after key expired and reset, got %d", rr.Code)
	}
}
func TestHandleClickSuspiciousAgentDetection(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()

	// disable background logger writes during unit evaluation to prevent race drops
	batchLogger = nil 

	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name: "upstream gateway sets automated tag explicit intercept",
			headers: map[string]string{
				"X-Click-Source": "automated",
				"User-Agent":     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "python runner default useragent trigger signature",
			headers: map[string]string{
				"User-Agent": "python-requests/2.28.2",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "terminal curl request client identification trigger",
			headers: map[string]string{
				"User-Agent": "curl/7.88.1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "absent client browser token headers classify as bot profile",
			headers: map[string]string{
				"User-Agent": "",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"campaign_id":"unit_test_verification"}`
			rr := performClickRequest(http.MethodPost, body, tt.headers)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status response %d, but received %d instead", tt.expectedStatus, rr.Code)
			}
		})
	}
}