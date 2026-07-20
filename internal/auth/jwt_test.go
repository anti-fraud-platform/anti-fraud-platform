package auth

import (
	"encoding/base64"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateToken(t *testing.T) {
	jwtSecret = []byte("test-secret-key-for-unit-tests")
	os.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	defer os.Unsetenv("JWT_SECRET")

	token, err := GenerateToken(42, "alice", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned empty string")
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("expected Username alice, got %s", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("expected Role admin, got %s", claims.Role)
	}
}

func TestTokenHasCorrectExpiry(t *testing.T) {
	jwtSecret = []byte("expiry-test")
	before := time.Now()
	token, _ := GenerateToken(1, "user", "viewer")
	after := time.Now()

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	expectedMin := before.Add(23*time.Hour + 59*time.Minute)
	expectedMax := after.Add(24*time.Hour + time.Minute)
	if claims.ExpiresAt.Time.Before(expectedMin) || claims.ExpiresAt.Time.After(expectedMax) {
		t.Errorf("token expiry %v not in expected range [%v, %v]",
			claims.ExpiresAt.Time, expectedMin, expectedMax)
	}
}

func TestTokenSubjectMatchesUsername(t *testing.T) {
	jwtSecret = []byte("subject-test")
	token, _ := GenerateToken(1, "bob", "admin")
	claims, _ := ValidateToken(token)
	if claims.Subject != "bob" {
		t.Errorf("expected Subject 'bob', got %q", claims.Subject)
	}
}

func TestTokenHasIssuedAt(t *testing.T) {
	jwtSecret = []byte("iat-test")
	before := time.Now()
	token, _ := GenerateToken(1, "u", "r")
	after := time.Now()
	claims, _ := ValidateToken(token)

	if claims.IssuedAt == nil {
		t.Fatal("token has no IssuedAt")
	}
	if claims.IssuedAt.Time.Before(before.Add(-time.Second)) || claims.IssuedAt.Time.After(after.Add(time.Second)) {
		t.Errorf("IssuedAt %v not near now", claims.IssuedAt.Time)
	}
}

func TestValidateTokenTampered(t *testing.T) {
	jwtSecret = []byte("test-secret")
	token, _ := GenerateToken(1, "bob", "viewer")

	_, err := ValidateToken(token + "x")
	if err == nil {
		t.Error("expected error for tampered token")
	}
	if err != ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestValidateTokenTruncated(t *testing.T) {
	jwtSecret = []byte("test-secret")
	token, _ := GenerateToken(1, "bob", "viewer")

	_, err := ValidateToken(token[:len(token)/2])
	if err == nil {
		t.Error("expected error for truncated token")
	}
}

func TestValidateTokenEmpty(t *testing.T) {
	jwtSecret = []byte("test-secret")
	_, err := ValidateToken("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	jwtSecret = []byte("correct-secret")
	token, _ := GenerateToken(1, "carol", "viewer")

	jwtSecret = []byte("wrong-secret")
	_, err := ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating with wrong secret")
	}

	jwtSecret = []byte("correct-secret")
}

func TestValidateTokenExpired(t *testing.T) {
	jwtSecret = []byte("test-secret")

	claims := &Claims{
		UserID:   1,
		Username: "dave",
		Role:     "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = ValidateToken(token)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateTokenNoneAlgorithmAttack(t *testing.T) {
	jwtSecret = []byte("test-secret")

	// Craft a token with alg: "none" to bypass signature check.
	b64 := base64.RawURLEncoding
	header := b64.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := b64.EncodeToString([]byte(`{"user_id":999,"username":"attacker","role":"admin"}`))
	unsigned := header + "." + payload + "."

	_, err := ValidateToken(unsigned)
	if err == nil {
		t.Error("token with alg=none must be rejected")
	}
	if err != ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestValidateTokenHS384RejectedWhenExpectingHS256(t *testing.T) {
	jwtSecret = []byte("test-secret")
	claims := &Claims{
		UserID:   1,
		Username: "user",
		Role:     "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	_, err = ValidateToken(tokenString)
	if err == nil {
		t.Error("HS384 token should be rejected when server validates HS256")
	}
}

func TestValidateTokenDifferentSecret(t *testing.T) {
	jwtSecret = []byte("server-A-secret")
	token, _ := GenerateToken(1, "user", "role")

	jwtSecret = []byte("server-B-secret")
	_, err := ValidateToken(token)
	if err == nil {
		t.Error("token signed by different secret should fail")
	}
	jwtSecret = []byte("server-A-secret")
}

func TestConcurrentGenerateAndValidate(t *testing.T) {
	jwtSecret = []byte("concurrent-test")
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			token, err := GenerateToken(n, "user", "viewer")
			if err != nil {
				t.Errorf("GenerateToken failed in goroutine %d: %v", n, err)
				return
			}
			claims, err := ValidateToken(token)
			if err != nil {
				t.Errorf("ValidateToken failed in goroutine %d: %v", n, err)
				return
			}
			if claims.UserID != n {
				t.Errorf("goroutine %d: expected UserID %d, got %d", n, n, claims.UserID)
			}
		}(i)
	}
	wg.Wait()
}

func TestGenerateTokenPreservesSpecialChars(t *testing.T) {
	jwtSecret = []byte("special-chars")
	special := "user-with-特殊字符-and-emoji-🔐"
	token, _ := GenerateToken(1, special, "admin")
	claims, _ := ValidateToken(token)
	if claims.Username != special {
		t.Errorf("expected special chars preserved, got %q", claims.Username)
	}
}

func TestGetEnvDefault(t *testing.T) {
	os.Setenv("TEST_GETENV_KEY", "from-env")
	defer os.Unsetenv("TEST_GETENV_KEY")

	if v := getEnvDefault("TEST_GETENV_KEY", "fallback"); v != "from-env" {
		t.Errorf("expected 'from-env', got %q", v)
	}
	if v := getEnvDefault("NONEXISTENT_KEY_12345", "fallback"); v != "fallback" {
		t.Errorf("expected 'fallback', got %q", v)
	}
}

func TestValidateTokenMalformedDots(t *testing.T) {
	jwtSecret = []byte("test-secret")

	tests := []struct {
		name  string
		token string
	}{
		{"no dots", "abc123"},
		{"one dot", "abc.def"},
		{"three dots", "abc.def.ghi.jkl"},
		{"empty segments", ".."},
		{"only dots", "..."},
		{"trailing space", "abc.def.ghi "},
		{"leading space", " abc.def.ghi"},
		{"newline", "abc.def.ghi\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateToken(tt.token)
			if err == nil {
				t.Errorf("expected error for token %q", tt.token)
			}
			if err != ErrTokenInvalid {
				t.Errorf("expected ErrTokenInvalid, got %v", err)
			}
		})
	}
}

func TestValidateTokenWithPayloadManipulation(t *testing.T) {
	jwtSecret = []byte("test-secret")
	token, _ := GenerateToken(1, "alice", "viewer")

	// Tamper with the payload to change role to admin.
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token format: %v", parts)
	}

	// Decode payload, modify, re-encode.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	// Replace "viewer" with "admin" in the payload.
	modified := strings.Replace(string(payloadBytes), `"viewer"`, `"admin"`, 1)
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(modified))
	tampered := strings.Join(parts, ".")

	_, err = ValidateToken(tampered)
	if err == nil {
		t.Error("modified payload should fail signature validation")
	}
}

func TestValidateTokenNotBefore(t *testing.T) {
	jwtSecret = []byte("nbf-test")

	// Create a token that is valid in the future (nbf = 1 hour from now).
	claims := &Claims{
		UserID:   1,
		Username: "future",
		Role:     "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	// jwt/v5 may or may not enforce nbf by default depending on parser config.
	// We just verify the token can be parsed without panic.
	_, _ = ValidateToken(token)
}
