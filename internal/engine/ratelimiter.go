package engine

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

type windowState struct {
	count     int
	expiresAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	maxRate int
	window  time.Duration
	clock   Clock
	clients map[string]*windowState
}

func NewRateLimiter(maxRate int, window time.Duration, clock Clock) *RateLimiter {
	return &RateLimiter{
		maxRate: maxRate,
		window:  window,
		clock:   clock,
		clients: make(map[string]*windowState),
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.clock.Now()
	state, exists := rl.clients[ip]

	if !exists || now.After(state.expiresAt) {
		rl.clients[ip] = &windowState{
			count:     1,
			expiresAt: now.Add(rl.window),
		}
		return true
	}

	if state.count >= rl.maxRate {
		return false
	}

	state.count++
	return true
}
