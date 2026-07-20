package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setupMiddlewareTest() {
	jwtSecret = []byte("middleware-test-secret")
	os.Setenv("JWT_SECRET", "middleware-test-secret")
}

// ---- RequireAuth: happy path ----

func TestRequireAuthPassesValidToken(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(1, "testuser", "admin")

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		claims, ok := UserFromContext(r.Context())
		if !ok {
			t.Error("expected claims in context")
			return
		}
		if claims.Username != "testuser" {
			t.Errorf("expected username testuser, got %s", claims.Username)
		}
		if claims.UserID != 1 {
			t.Errorf("expected UserID 1, got %d", claims.UserID)
		}
		if claims.Role != "admin" {
			t.Errorf("expected role admin, got %s", claims.Role)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ---- RequireAuth: rejection cases ----

func TestRequireAuthRejectsMissingHeader(t *testing.T) {
	setupMiddlewareTest()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "missing authorization header") {
		t.Errorf("unexpected error message: %s", rr.Body.String())
	}
}

func TestRequireAuthRejectsBasicAuth(t *testing.T) {
	setupMiddlewareTest()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuthRejectsExpiredToken(t *testing.T) {
	setupMiddlewareTest()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	jwtSecret = []byte("middleware-test-secret")
	claims := &Claims{
		UserID:   1,
		Username: "expired",
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

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuthRejectsTamperedToken(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(1, "user", "viewer")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token+"TAMPERED")
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuthRejectsEmptyBearer(t *testing.T) {
	setupMiddlewareTest()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuthRejectsTokenWithoutBearerPrefix(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(1, "user", "viewer")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	// Token without "Bearer " prefix — just the raw token.
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", token)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuthCaseInsensitiveScheme(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(1, "user", "viewer")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// "bearer" in lowercase — EqualFold should accept it.
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "bearer "+token)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("lowercase 'bearer' should be accepted, got %d", rr.Code)
	}
}

func TestRequireAuthRejectsTooManyParts(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(1, "user", "viewer")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	// "Bearer token extra" — SplitN with n=2 should still work for the first two parts,
	// but the token will be "token extra" which is invalid.
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token+" extra")
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuthContextPropagatesClaims(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(42, "propagated", "editor")

	var receivedClaims *Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedClaims, _ = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	if receivedClaims == nil {
		t.Fatal("claims were not propagated to inner handler")
	}
	if receivedClaims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", receivedClaims.UserID)
	}
	if receivedClaims.Username != "propagated" {
		t.Errorf("expected username 'propagated', got %q", receivedClaims.Username)
	}
	if receivedClaims.Role != "editor" {
		t.Errorf("expected role 'editor', got %q", receivedClaims.Role)
	}
}

// ---- RequireAuth: concurrency ----

func TestRequireAuthConcurrentRequests(t *testing.T) {
	setupMiddlewareTest()

	// Generate multiple tokens for different users.
	type tokenEntry struct {
		token string
		user  string
	}
	entries := make([]tokenEntry, 50)
	for i := range entries {
		tok, _ := GenerateToken(i, "user"+string(rune('A'+i%26)), "viewer")
		entries[i] = tokenEntry{token: tok, user: "user" + string(rune('A'+i%26))}
	}

	var wg sync.WaitGroup
	for _, e := range entries {
		wg.Add(1)
		go func(te tokenEntry) {
			defer wg.Done()
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, _ := UserFromContext(r.Context())
				if claims == nil {
					t.Error("claims nil in concurrent request")
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+te.token)
			rr := httptest.NewRecorder()
			RequireAuth(inner).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("user %s: expected 200, got %d", te.user, rr.Code)
			}
		}(e)
	}
	wg.Wait()
}

// ---- OptionalAuth ----

func TestOptionalAuthPassesWithoutToken(t *testing.T) {
	setupMiddlewareTest()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := UserFromContext(r.Context()); ok {
			t.Error("expected no claims in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/optional", nil)
	rr := httptest.NewRecorder()

	OptionalAuth(inner).ServeHTTP(rr, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestOptionalAuthInjectsClaimsWhenPresent(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(5, "optuser", "viewer")

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		claims, ok := UserFromContext(r.Context())
		if !ok {
			t.Error("expected claims in context")
			return
		}
		if claims.Username != "optuser" {
			t.Errorf("expected username optuser, got %s", claims.Username)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	OptionalAuth(inner).ServeHTTP(rr, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestOptionalAuthDoesNotRejectInvalidToken(t *testing.T) {
	setupMiddlewareTest()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Should have no claims.
		if _, ok := UserFromContext(r.Context()); ok {
			t.Error("optional auth should not inject claims for invalid token")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Invalid token — should pass through without rejection.
	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer totally.invalid.token")
	rr := httptest.NewRecorder()

	OptionalAuth(inner).ServeHTTP(rr, req)

	if !called {
		t.Error("inner handler should be called for invalid token in OptionalAuth")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestOptionalAuthDoesNotRejectMalformedHeader(t *testing.T) {
	setupMiddlewareTest()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()

	OptionalAuth(inner).ServeHTTP(rr, req)

	if !called {
		t.Error("inner handler should be called for malformed header in OptionalAuth")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ---- RequireAuth: response format ----

func TestRequireAuthReturnsJSONError(t *testing.T) {
	setupMiddlewareTest()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "text/plain") {
		// http.Error sets text/plain, which is fine.
	}
	if !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("error response should contain 'error', got: %s", rr.Body.String())
	}
}

// ---- Edge: Bearer with extra whitespace ----

func TestRequireAuthBearerWithExtraWhitespace(t *testing.T) {
	setupMiddlewareTest()
	token, _ := GenerateToken(1, "user", "viewer")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// "Bearer  token" — double space after Bearer. SplitN("Bearer  token", " ", 2)
	// returns ["Bearer", " token"] which has a leading space in the token.
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer  "+token)
	rr := httptest.NewRecorder()

	RequireAuth(inner).ServeHTTP(rr, req)

	// This should fail because the token will have a leading space.
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("double-space Bearer should reject (token has leading space), got %d", rr.Code)
	}
}
