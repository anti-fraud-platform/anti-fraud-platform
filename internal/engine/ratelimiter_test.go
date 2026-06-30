package engine

import (
	"testing"
	"time"
)

type FakeClock struct {
	wallTime time.Time
}

func (f *FakeClock) Now() time.Time {
	return f.wallTime
}

func (f *FakeClock) Advance(d time.Duration) {
	f.wallTime = f.wallTime.Add(d)
}

func TestRateLimiter(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := &FakeClock{wallTime: startTime}

	maxRate := 5
	window := time.Second
	rl := NewRateLimiter(maxRate, window, fakeClock)

	ip := "192.168.1.1"

	if !rl.Allow(ip) {
		t.Errorf("Expected the very first request to be allowed")
	}

	for i := 2; i <= maxRate; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Expected request %d to be allowed (under/at the edge of threshold)", i)
		}
	}

	if rl.Allow(ip) {
		t.Errorf("Expected request %d to be blocked (strictly over threshold)", maxRate+1)
	}

	fakeClock.Advance(window + time.Millisecond)

	if !rl.Allow(ip) {
		t.Errorf("Expected request to be allowed after the time window has passed and counter reset")
	}
}
