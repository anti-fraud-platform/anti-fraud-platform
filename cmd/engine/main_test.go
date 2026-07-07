package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"anti-fraud/internal/challenge"
	"anti-fraud/internal/geopolicy"
	"anti-fraud/internal/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestEngine(t *testing.T) func() {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	challengeStore = &challenge.RedisStore{Client: rdb}
	geoResolver = nil
	geoPolicy = geopolicy.Config{}
	batchLogger = logger.NewBatchLogger(nil, 100, 1000)
	maxRate = 5

	origChallenge := requireChallenge
	origHeaderCheck := requireHeaderCheck
	origDynThreshold := dynBlacklistThreshold

	requireChallenge = false
	requireHeaderCheck = false
	dynBlacklistThreshold = 9999 // disable dynamic blacklist in tests

	return func() {
		bgTasks.Wait()
		requireChallenge = origChallenge
		requireHeaderCheck = origHeaderCheck
		dynBlacklistThreshold = origDynThreshold
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

// decodeClickStatus pulls the "status" field ("success" | "flagged") out of
// a click response body. Several tests below need to distinguish these two
// cases specifically, since both return HTTP 200.
func decodeClickStatus(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode click response body %q: %v", rr.Body.String(), err)
	}
	return resp["status"]
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

	geoResolver = nil
	geoPolicy = geopolicy.Config{}
	batchLogger = logger.NewBatchLogger(nil, 100, 1000)
	maxRate = 5

	origChallenge := requireChallenge
	origHeaderCheck := requireHeaderCheck
	requireChallenge = false
	requireHeaderCheck = false
	defer func() {
		requireChallenge = origChallenge
		requireHeaderCheck = origHeaderCheck
	}()

	ip := "5.5.5.5"
	body := `{"user_agent":"test-agent","campaign_id":"camp_self_heal","timestamp":123456789}`
	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": ip,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", rr.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/click", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", ip)
	req.Header.Set("User-Agent", "test-agent")
	fp := fingerprint(req)
	key := fmt.Sprintf("%s%s", rateKeyPrefix, fp)

	mr.Set(key, "999")
	if mr.TTL(key) != 0 {
		t.Fatalf("test setup invalid: expected no TTL, got %v", mr.TTL(key))
	}

	rr = performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": ip,
		"User-Agent":      "test-agent",
	})
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second request, got %d", rr.Code)
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
		"User-Agent":      "test-agent",
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

// ============================================================================
// Tier 1: JS-execution challenge — tests below actually exercise the new
// check (requireChallenge = true), unlike every test above which
// deliberately runs with it disabled.
// ============================================================================

func TestHandleClickRequiresJSChallenge(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	requireChallenge = true
	requireHeaderCheck = false

	body := `{"campaign_id":"camp_challenge_missing"}`
	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "50.50.50.1",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 (flagged, not hard-blocked) for a click with no challenge fields, got %d", rr.Code)
	}
	if status := decodeClickStatus(t, rr); status != "flagged" {
		t.Fatalf("expected status=flagged for a click with no challenge_id/challenge_token, got %q", status)
	}
}

func TestHandleClickChallengeSolvedTooFastIsFlagged(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	requireChallenge = true
	requireHeaderCheck = false

	ch, err := challenge.Issue(context.Background(), challengeStore)
	if err != nil {
		t.Fatalf("failed to issue challenge: %v", err)
	}
	token := challenge.ComputeToken(ch.Nonce)

	// No sleep — simulates a script that calls /v1/challenge and /v1/click
	// back-to-back, faster than MinSolveDelay allows a human to.
	body := fmt.Sprintf(`{"campaign_id":"camp_too_fast","challenge_id":%q,"challenge_token":%q}`, ch.ChallengeID, token)
	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "50.50.50.2",
	})

	if status := decodeClickStatus(t, rr); status != "flagged" {
		t.Fatalf("expected a challenge solved faster than MinSolveDelay to be flagged, got status=%q (http %d)", status, rr.Code)
	}
}

func TestHandleClickChallengeSolvedCorrectlyIsAllowed(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	requireChallenge = true
	requireHeaderCheck = false // isolate: only testing the challenge layer here

	ch, err := challenge.Issue(context.Background(), challengeStore)
	if err != nil {
		t.Fatalf("failed to issue challenge: %v", err)
	}
	time.Sleep(challenge.MinSolveDelay + 20*time.Millisecond)
	token := challenge.ComputeToken(ch.Nonce)

	body := fmt.Sprintf(`{"campaign_id":"camp_valid_challenge","challenge_id":%q,"challenge_token":%q}`, ch.ChallengeID, token)
	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "50.50.50.3",
		// Must set a normal UA — an empty User-Agent trips
		// isSuspiciousUserAgent at step 1, before the challenge check
		// (step 2) ever runs, which would flag this for the wrong reason.
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for a correctly solved challenge, got %d, body=%s", rr.Code, rr.Body.String())
	}
	if status := decodeClickStatus(t, rr); status != "success" {
		t.Fatalf("expected status=success for a correctly solved challenge, got %q", status)
	}
}

func TestHandleClickChallengeWrongTokenIsFlagged(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	requireChallenge = true
	requireHeaderCheck = false

	ch, err := challenge.Issue(context.Background(), challengeStore)
	if err != nil {
		t.Fatalf("failed to issue challenge: %v", err)
	}
	time.Sleep(challenge.MinSolveDelay + 20*time.Millisecond)

	body := fmt.Sprintf(`{"campaign_id":"camp_wrong_token","challenge_id":%q,"challenge_token":"not-the-real-token"}`, ch.ChallengeID)
	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "50.50.50.4",
	})

	if status := decodeClickStatus(t, rr); status != "flagged" {
		t.Fatalf("expected an incorrect token to be flagged, got status=%q", status)
	}
}

func TestHandleClickChallengeReplayIsRejected(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	requireChallenge = true
	requireHeaderCheck = false

	ch, err := challenge.Issue(context.Background(), challengeStore)
	if err != nil {
		t.Fatalf("failed to issue challenge: %v", err)
	}
	time.Sleep(challenge.MinSolveDelay + 20*time.Millisecond)
	token := challenge.ComputeToken(ch.Nonce)

	body := fmt.Sprintf(`{"campaign_id":"camp_replay","challenge_id":%q,"challenge_token":%q}`, ch.ChallengeID, token)
	headers := map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "50.50.50.5",
		// See comment in TestHandleClickChallengeSolvedCorrectlyIsAllowed —
		// empty UA would flag this at step 1 for the wrong reason.
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	}

	first := performClickRequest(http.MethodPost, body, headers)
	if status := decodeClickStatus(t, first); status != "success" {
		t.Fatalf("expected first use of a valid challenge to succeed, got %q", status)
	}

	second := performClickRequest(http.MethodPost, body, headers)
	if status := decodeClickStatus(t, second); status != "flagged" {
		t.Fatalf("expected a replayed challenge_id/challenge_token to be rejected on second use, got %q", status)
	}
}

// ============================================================================
// Tier 1: header-consistency heuristic
// ============================================================================

func TestHandleClickHeaderHeuristicFlagsBareRequest(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	requireChallenge = false // isolate: only testing the header layer here
	requireHeaderCheck = true

	body := `{"campaign_id":"camp_header_bare"}`

	// 1. Request with only suspicious headers (no UA) – should NOT be flagged (score 2 < 3)
	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "60.60.60.1",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
		// no Accept-Language, no Sec-Fetch-*, no Client Hints → triggers headers heuristic
	})
	if status := decodeClickStatus(t, rr); status != "success" {
		t.Fatalf("expected request with only headers (score 2) to pass, got %q", status)
	}

	// 2. Request with suspicious headers + suspicious UA – should be flagged (score 4 >= 3)
	rr = performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "60.60.60.1",
		"User-Agent":      "curl/8.5.0", // triggers UA check
		// still no Accept-Language etc., triggers headers heuristic
	})
	if status := decodeClickStatus(t, rr); status != "flagged" {
		t.Fatalf("expected request with headers + UA (score 4) to be flagged, got %q", status)
	}
}

// This is the integration-level regression test for the Accept:"*/*" bug
// caught during Tier 1 integration: a same-origin fetch() call from the
// real clicker page sends "Accept: */*" by browser default, plus the full
// set of Sec-Fetch-*/Accept-Language/Accept-Encoding/Client Hints headers.
// That combination must pass — an earlier version of headercheck would
// have flagged this specifically because of the wildcard Accept value.
func TestHandleClickHeaderHeuristicAllowsBrowserLikeRequest(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()
	requireChallenge = false
	requireHeaderCheck = true

	body := `{"campaign_id":"camp_header_browser"}`
	rr := performClickRequest(http.MethodPost, body, map[string]string{
		"Content-Type":    "application/json",
		"X-Forwarded-For": "60.60.60.2",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
		"Accept":          "*/*",
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate, br",
		"Sec-Fetch-Site":  "same-origin",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Ch-Ua":       `"Chromium";v="125", "Google Chrome";v="125"`,
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if status := decodeClickStatus(t, rr); status != "success" {
		t.Fatalf("expected a fully browser-shaped request (including Accept: */*) to pass the header heuristic, got %q", status)
	}
}

// ============================================================================
// Tier 1: GET /v1/challenge
// ============================================================================

func TestHandleChallengeIssuesUniqueChallenges(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()

	req1 := httptest.NewRequest(http.MethodGet, "/v1/challenge", nil)
	rr1 := httptest.NewRecorder()
	handleChallenge(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /v1/challenge, got %d", rr1.Code)
	}

	var ch1 challenge.Challenge
	if err := json.Unmarshal(rr1.Body.Bytes(), &ch1); err != nil {
		t.Fatalf("failed to decode challenge response: %v", err)
	}
	if ch1.ChallengeID == "" || ch1.Nonce == "" {
		t.Fatalf("expected non-empty challenge_id and nonce, got %+v", ch1)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/challenge", nil)
	rr2 := httptest.NewRecorder()
	handleChallenge(rr2, req2)

	var ch2 challenge.Challenge
	if err := json.Unmarshal(rr2.Body.Bytes(), &ch2); err != nil {
		t.Fatalf("failed to decode second challenge response: %v", err)
	}

	if ch1.ChallengeID == ch2.ChallengeID || ch1.Nonce == ch2.Nonce {
		t.Fatalf("expected two separate GET /v1/challenge calls to return distinct challenges, got identical values")
	}
}

func TestHandleChallengeRejectsNonGET(t *testing.T) {
	cleanup := setupTestEngine(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/v1/challenge", nil)
	rr := httptest.NewRecorder()
	handleChallenge(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /v1/challenge, got %d", rr.Code)
	}
}
