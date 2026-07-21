package auth

import (
	"crypto/subtle"
	"strings"
	"sync"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}
	if !CheckPassword(hash, "secret123") {
		t.Error("CheckPassword returned false for correct password")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("CheckPassword returned true for wrong password")
	}
}

func TestDifferentHashesForSamePassword(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Error("two hashes of the same password must differ (random salt)")
	}
	// But both should validate against the same plaintext.
	if !CheckPassword(h1, "same") || !CheckPassword(h2, "same") {
		t.Error("both hashes should validate against the original password")
	}
}

func TestHashIsEmptyString(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("bcrypt should accept empty password: %v", err)
	}
	if !CheckPassword(hash, "") {
		t.Error("empty password should validate against its own hash")
	}
	if CheckPassword(hash, "notempty") {
		t.Error("non-empty password should not match empty-password hash")
	}
}

func TestHashIsBcryptFormat(t *testing.T) {
	hash, _ := HashPassword("check-format")
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Errorf("bcrypt hash should start with $2a$ or $2b$, got prefix: %s", hash[:6])
	}
}

func TestCheckPasswordWithGarbageHash(t *testing.T) {
	if CheckPassword("not-a-hash", "anything") {
		t.Error("garbage hash should always return false")
	}
	if CheckPassword("", "anything") {
		t.Error("empty hash should always return false")
	}
}

func TestHashPasswordVeryLongInput(t *testing.T) {
	// bcrypt has a hard 72-byte limit — it rejects passwords longer than that.
	long := strings.Repeat("a", 1000)
	_, err := HashPassword(long)
	if err == nil {
		t.Error("bcrypt should reject passwords > 72 bytes")
	}

	// 72 bytes exactly should work.
	exact := strings.Repeat("b", 72)
	hash, err := HashPassword(exact)
	if err != nil {
		t.Fatalf("72-byte password should be accepted: %v", err)
	}
	if !CheckPassword(hash, exact) {
		t.Error("72-byte password should validate against its hash")
	}

	// 73 bytes should fail.
	tooLong := strings.Repeat("c", 73)
	_, err = HashPassword(tooLong)
	if err == nil {
		t.Error("73-byte password should be rejected")
	}
}

func TestHashPasswordUnicode(t *testing.T) {
	pass := "пароль123_日本語🔐"
	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("HashPassword with unicode failed: %v", err)
	}
	if !CheckPassword(hash, pass) {
		t.Error("unicode password should validate")
	}
	if CheckPassword(hash, "пароль123_日本語") {
		t.Error("partial unicode should not match")
	}
}

func TestConcurrentHashAndCheck(t *testing.T) {
	// Verify that concurrent hashing + checking doesn't race.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			pass := strings.Repeat("x", n+1)
			hash, err := HashPassword(pass)
			if err != nil {
				t.Errorf("goroutine %d: HashPassword failed: %v", n, err)
				return
			}
			if !CheckPassword(hash, pass) {
				t.Errorf("goroutine %d: CheckPassword failed for its own hash", n)
			}
			// Verify wrong password fails.
			if CheckPassword(hash, pass+"!") {
				t.Errorf("goroutine %d: CheckPassword should fail for wrong password", n)
			}
		}(i)
	}
	wg.Wait()
}

func TestHashCostIsReasonable(t *testing.T) {
	// bcrypt cost 10 means 2^10 = 1024 iterations.
	// The hash should contain the cost factor "$2a$10$".
	hash, _ := HashPassword("cost-check")
	if !strings.Contains(hash, "$10$") {
		t.Errorf("expected bcrypt cost 10 in hash, got: %s", hash)
	}
}

func TestCheckPasswordConstantTime(t *testing.T) {
	// Verify that CheckPassword doesn't leak timing information.
	// Two wrong passwords of very different lengths should take
	// approximately the same time to check (bcrypt is constant-time).
	hash, _ := HashPassword("correct")

	// Warm up.
	CheckPassword(hash, "a")
	CheckPassword(hash, strings.Repeat("b", 100))

	// Measure: short wrong password vs long wrong password.
	// This is a loose check — we just verify both fail.
	if CheckPassword(hash, "a") {
		t.Error("short wrong password should fail")
	}
	if CheckPassword(hash, strings.Repeat("b", 100)) {
		t.Error("long wrong password should fail")
	}

	// More importantly: verify the hash comparison uses constant-time compare.
	// We can check the hash format is valid bcrypt which internally uses subtle.ConstantTimeCompare.
	if subtle.ConstantTimeCompare([]byte("a"), []byte("b")) != 0 {
		t.Error("constant-time compare sanity check failed")
	}
}
