// Package challenge implements a JS-execution proof-of-work check for the
// click endpoint. A bare curl/python-requests/Go-http-client call cannot
// pass it, because it requires calling GET /v1/challenge first and
// computing a hash client-side (in JS) before the click is accepted.
//
// Honest limitation, worth stating in the report: this is not unbeatable.
// A motivated attacker can read this file and reimplement ComputeToken in
// any language. What it rules out is the naive case — a script that POSTs
// to /v1/click with zero awareness of the challenge flow — which is the gap
// the old X-Click-Source-based bot simulation never actually exercised.
package challenge

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Salt is a fixed, non-secret string mixed into the token hash. It is not a
// security secret — its only purpose is to make the client actually perform
// the same hashing step (in JS) rather than echoing something back.
const Salt = "af-js-check-v1"

// TTL is how long a challenge stays valid after issuance.
const TTL = 60 * time.Second

// MinSolveDelay is the minimum time that must elapse between issuing a
// challenge and receiving a solved click. A human opening the page and
// clicking a button cannot do this in under ~150ms; a script that issues
// the challenge and immediately answers it can.
const MinSolveDelay = 150 * time.Millisecond

var (
	ErrNotFound = errors.New("challenge: not found or expired")
	ErrTooFast  = errors.New("challenge: solved too fast")
	ErrMismatch = errors.New("challenge: token mismatch")
)

// Store is the minimal key/value interface challenge needs. See
// redis_store.go for a *redis.Client adapter — wire it to the SAME
// *redis.Client the engine already uses for rate limiting (var rdb in
// cmd/engine/main.go), don't open a second connection pool.
type Store interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
}

type record struct {
	Nonce    string `json:"nonce"`
	IssuedAt int64  `json:"issued_at_ms"`
}

// Challenge is the JSON payload returned to the client from GET /v1/challenge.
type Challenge struct {
	ChallengeID string `json:"challenge_id"`
	Nonce       string `json:"nonce"`
	IssuedAtMS  int64  `json:"issued_at"`
}

// Issue creates and stores a new challenge, returning it for the caller to
// serialize as the HTTP response body of GET /v1/challenge.
func Issue(ctx context.Context, store Store) (*Challenge, error) {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("generate challenge id: %w", err)
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	id := hex.EncodeToString(idBytes)
	nonce := hex.EncodeToString(nonceBytes)
	issuedAt := time.Now().UnixMilli()

	rec := record{Nonce: nonce, IssuedAt: issuedAt}
	payload, err := json.Marshal(rec)
	if err != nil {
		return nil, fmt.Errorf("marshal challenge record: %w", err)
	}

	if err := store.Set(ctx, key(id), string(payload), TTL); err != nil {
		return nil, fmt.Errorf("store challenge: %w", err)
	}

	return &Challenge{ChallengeID: id, Nonce: nonce, IssuedAtMS: issuedAt}, nil
}

// Validate checks a client-submitted (challengeID, token) pair. On success
// the challenge is deleted (single use) and nil is returned. On failure it
// returns a sentinel error identifying why, so the caller can log a
// specific click_logs.reason value:
//
//	ErrNotFound -> "no_js_challenge"     (missing, expired, or replayed)
//	ErrTooFast  -> "challenge_too_fast"
//	ErrMismatch -> "challenge_mismatch"
func Validate(ctx context.Context, store Store, challengeID, token string) error {
	if challengeID == "" || token == "" {
		return ErrNotFound
	}

	raw, err := store.Get(ctx, key(challengeID))
	if err != nil || raw == "" {
		return ErrNotFound
	}
	// Best-effort single use; TTL cleans up regardless if this fails.
	_ = store.Del(ctx, key(challengeID))

	var rec record
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return ErrNotFound
	}

	elapsed := time.Since(time.UnixMilli(rec.IssuedAt))
	if elapsed < MinSolveDelay {
		return ErrTooFast
	}

	if ComputeToken(rec.Nonce) != token {
		return ErrMismatch
	}

	return nil
}

// ComputeToken mirrors the browser-side JS computation exactly:
//
//	sha256(nonce + ":" + Salt), hex-encoded lowercase.
//
// See deployments/nginx/clicker page JS for the matching client-side code,
// and cmd/generator for the matching Go implementation used by the
// "smart bot" traffic profile.
func ComputeToken(nonce string) string {
	sum := sha256.Sum256([]byte(nonce + ":" + Salt))
	return hex.EncodeToString(sum[:])
}

func key(challengeID string) string {
	return "af:challenge:" + challengeID
}
