package engine

import (
	"testing"
	"time"
)

type fakeClock struct {
	wallTime time.Time
}

func (f *fakeClock) Now() time.Time {
	return f.wallTime
}

func (f *fakeClock) Advance(d time.Duration) {
	f.wallTime = f.wallTime.Add(d)
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := &fakeClock{wallTime: startTime}

	limiter := NewRateLimiter(5, time.Second, clock)
	ip := "192.168.1.1"

	for i := 0; i < 5; i++ {
		if !limiter.Allow(ip) {
			t.Fatalf("expected request %d to pass inside the current window", i+1)
		}
	}

	if limiter.Allow(ip) {
		t.Fatal("expected request 6 to be blocked")
	}

	clock.Advance(time.Second + time.Millisecond)

	if !limiter.Allow(ip) {
		t.Fatal("expected limiter to reset after the window expires")
	}
}
