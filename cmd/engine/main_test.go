package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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
				"ip":"` + tt.ip + `",
				"user_agent":"test-agent",
				"campaign_id":"camp_rate_limit",
				"timestamp":123456789
			}`

			for i := 0; i < tt.requestsToSend; i++ {
				rr := performClickRequest(http.MethodPost, body, map[string]string{
					"Content-Type": "application/json",
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
			"ip":"10.0.0.` + strconv.Itoa(i+1) + `",
			"user_agent":"test-agent",
			"campaign_id":"camp_unique_ips",
			"timestamp":123456789
		}`

		rr := performClickRequest(http.MethodPost, body, map[string]string{
			"Content-Type": "application/json",
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("expected different IP request to stay 200, got %d, body: %s", rr.Code, rr.Body.String())
		}
	}
}
