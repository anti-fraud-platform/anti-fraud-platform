package challenge

import (
	"context"
	"sync"
	"testing"
	"time"
)

// memStore is a trivial in-memory Store for tests — no real Redis needed.
type memStore struct {
	mu   sync.Mutex
	data map[string]string
}

func newMemStore() *memStore { return &memStore{data: map[string]string{}} }

func (m *memStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *memStore) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[key], nil
}

func (m *memStore) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func TestIssueThenValidate_Success(t *testing.T) {
	store := newMemStore()
	ctx := context.Background()

	ch, err := Issue(ctx, store)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	time.Sleep(MinSolveDelay + 10*time.Millisecond)

	token := ComputeToken(ch.Nonce)
	if err := Validate(ctx, store, ch.ChallengeID, token); err != nil {
		t.Fatalf("Validate: expected success, got %v", err)
	}
}

func TestValidate_MissingChallenge(t *testing.T) {
	err := Validate(context.Background(), newMemStore(), "does-not-exist", "whatever")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestValidate_WrongToken(t *testing.T) {
	store := newMemStore()
	ctx := context.Background()
	ch, _ := Issue(ctx, store)
	time.Sleep(MinSolveDelay + 10*time.Millisecond)

	if err := Validate(ctx, store, ch.ChallengeID, "not-the-real-token"); err != ErrMismatch {
		t.Fatalf("expected ErrMismatch, got %v", err)
	}
}

func TestValidate_TooFast(t *testing.T) {
	// The case that matters most for the demo: a script that calls
	// /v1/challenge and /v1/click back-to-back, faster than any human
	// clicking a button — even if it computed the token correctly.
	store := newMemStore()
	ctx := context.Background()
	ch, _ := Issue(ctx, store)

	token := ComputeToken(ch.Nonce)
	if err := Validate(ctx, store, ch.ChallengeID, token); err != ErrTooFast {
		t.Fatalf("expected ErrTooFast, got %v", err)
	}
}

func TestValidate_SingleUse(t *testing.T) {
	store := newMemStore()
	ctx := context.Background()
	ch, _ := Issue(ctx, store)
	time.Sleep(MinSolveDelay + 10*time.Millisecond)
	token := ComputeToken(ch.Nonce)

	if err := Validate(ctx, store, ch.ChallengeID, token); err != nil {
		t.Fatalf("first validate should succeed: %v", err)
	}
	if err := Validate(ctx, store, ch.ChallengeID, token); err != ErrNotFound {
		t.Fatalf("replayed token should fail with ErrNotFound, got %v", err)
	}
}

func TestValidate_ExpiredOrMissingChallenge(t *testing.T) {
	store := newMemStore()
	err := Validate(context.Background(), store, "expired-id", ComputeToken("whatever-nonce"))
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for expired/missing challenge, got %v", err)
	}
}
