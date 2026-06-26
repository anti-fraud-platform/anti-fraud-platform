package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestClickIntegrationPipeline(t *testing.T) {
	ctx := context.Background()
	rdb = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6380",
	})

	testIP := "123.45.67.89"
	rdb.Del(ctx, "rate:"+testIP)

	maxRate = 5

	ts := httptest.NewServer(http.HandlerFunc(handleClick))
	defer ts.Close()

	client := &http.Client{}

	payload := []byte(`{"campaign_id": "test_campaign", "user_agent": "Mozilla/5.0"}`)
	req, _ := http.NewRequest("POST", ts.URL+"/v1/click", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", testIP) // Подсовываем наш чистый IP

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
