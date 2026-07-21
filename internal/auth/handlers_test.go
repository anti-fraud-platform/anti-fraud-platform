package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

// ---------- Unit tests (no DB required) ----------

func TestRegisterRejectsNonPOST(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	for _, method := range []string{"GET", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/v1/auth/register", nil)
			rr := httptest.NewRecorder()
			handlers.RegisterHandler(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", rr.Code)
			}
		})
	}
}

func TestRegisterRejectsEmptyBody(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	req := httptest.NewRequest("POST", "/v1/auth/register", nil)
	rr := httptest.NewRecorder()
	handlers.RegisterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for nil body, got %d", rr.Code)
	}
}

func TestRegisterRejectsInvalidJSON(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader([]byte("{bad json")))
	rr := httptest.NewRecorder()
	handlers.RegisterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestRegisterRejectsMissingFields(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	tests := []struct {
		name string
		body string
	}{
		{"missing username", `{"password":"pass123"}`},
		{"missing password", `{"username":"validuser"}`},
		{"empty body", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handlers.RegisterHandler(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRegisterRejectsShortUsername(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	body, _ := json.Marshal(map[string]string{"username": "ab", "password": "pass123"})
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handlers.RegisterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for username < 3 chars, got %d", rr.Code)
	}
}

func TestRegisterRejectsLongUsername(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	long := strings.Repeat("a", 65)
	body, _ := json.Marshal(map[string]string{"username": long, "password": "pass123"})
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handlers.RegisterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for username > 64 chars, got %d", rr.Code)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	body, _ := json.Marshal(map[string]string{"username": "validuser", "password": "123"})
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handlers.RegisterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for password < 6 chars, got %d", rr.Code)
	}
}

func TestLoginRejectsNonPOST(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	for _, method := range []string{"GET", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/v1/auth/login", nil)
			rr := httptest.NewRecorder()
			handlers.LoginHandler(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", rr.Code)
			}
		})
	}
}

func TestLoginRejectsEmptyBody(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	req := httptest.NewRequest("POST", "/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	handlers.LoginHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for nil body, got %d", rr.Code)
	}
}

func TestLoginRejectsInvalidJSON(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	req := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()
	handlers.LoginHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestLoginRejectsMissingFields(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	tests := []struct {
		name string
		body string
	}{
		{"missing username", `{"password":"pass"}`},
		{"missing password", `{"username":"user"}`},
		{"empty body", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handlers.LoginHandler(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestMeRejectsNonGET(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/v1/auth/me", nil)
			rr := httptest.NewRecorder()
			handlers.MeHandler(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", rr.Code)
			}
		})
	}
}

func TestMeRejectsWithoutAuthContext(t *testing.T) {
	handlers := NewAuthHandlers(nil)

	req := httptest.NewRequest("GET", "/v1/auth/me", nil)
	rr := httptest.NewRecorder()
	handlers.MeHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when no context claims, got %d", rr.Code)
	}
}

func TestMeReturnsClaimsFromContext(t *testing.T) {
	jwtSecret = []byte("me-test")
	handlers := NewAuthHandlers(nil)

	claims := &Claims{UserID: 7, Username: "meuser", Role: "editor"}
	ctx := ContextWithUser(context.Background(), claims)

	req := httptest.NewRequest("GET", "/v1/auth/me", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handlers.MeHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp userResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != 7 {
		t.Errorf("expected ID 7, got %d", resp.ID)
	}
	if resp.Username != "meuser" {
		t.Errorf("expected username 'meuser', got %q", resp.Username)
	}
	if resp.Role != "editor" {
		t.Errorf("expected role 'editor', got %q", resp.Role)
	}
}

func TestMeResponseContentType(t *testing.T) {
	jwtSecret = []byte("me-format-test")
	handlers := NewAuthHandlers(nil)

	claims := &Claims{UserID: 1, Username: "u", Role: "r"}
	ctx := ContextWithUser(context.Background(), claims)

	req := httptest.NewRequest("GET", "/v1/auth/me", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handlers.MeHandler(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// ---------- Integration tests (require live Postgres) ----------

func setupIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	os.Setenv("JWT_SECRET", "integration-test-secret")
	jwtSecret = []byte("integration-test-secret")

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "antifraud"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "antifraud123"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "analytics"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
		return nil
	}
	if err = db.Ping(); err != nil {
		t.Skipf("skipping integration test: database not reachable: %v", err)
		return nil
	}
	return db
}

func TestIntegrationRegisterLoginMeFlow(t *testing.T) {
	db := setupIntegrationDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer db.Exec("DELETE FROM users WHERE username = $1", "testflow_e2e")

	store := NewUserStore(db)
	handlers := NewAuthHandlers(store)

	// 1) Register.
	body, _ := json.Marshal(registerRequest{Username: "testflow_e2e", Password: "pass123456"})
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handlers.RegisterHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2) Duplicate register → 409.
	req = httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handlers.RegisterHandler(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d: %s", rr.Code, rr.Body.String())
	}

	// 3) Login with correct credentials.
	loginBody, _ := json.Marshal(loginRequest{Username: "testflow_e2e", Password: "pass123456"})
	req = httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handlers.LoginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var tokenResp tokenResponse
	json.Unmarshal(rr.Body.Bytes(), &tokenResp)
	if tokenResp.Token == "" {
		t.Fatal("login: expected non-empty token")
	}

	// 4) Wrong password → 401.
	wrongBody, _ := json.Marshal(loginRequest{Username: "testflow_e2e", Password: "wrongpassword"})
	req = httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(wrongBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handlers.LoginHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: expected 401, got %d: %s", rr.Code, rr.Body.String())
	}

	// 5) Nonexistent user → 401.
	ghostBody, _ := json.Marshal(loginRequest{Username: "ghost_user_xyz", Password: "pass123"})
	req = httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(ghostBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handlers.LoginHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("ghost login: expected 401, got %d: %s", rr.Code, rr.Body.String())
	}

	// 6) /me with valid token through RequireAuth.
	req = httptest.NewRequest("GET", "/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.Token)
	rr = httptest.NewRecorder()
	RequireAuth(http.HandlerFunc(handlers.MeHandler)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var meResp userResponse
	json.Unmarshal(rr.Body.Bytes(), &meResp)
	if meResp.Username != "testflow_e2e" {
		t.Errorf("me: expected username 'testflow_e2e', got %q", meResp.Username)
	}
	if meResp.Role != "viewer" {
		t.Errorf("me: expected role 'viewer', got %q", meResp.Role)
	}

	// 7) /me without token → 401.
	req = httptest.NewRequest("GET", "/v1/auth/me", nil)
	rr = httptest.NewRecorder()
	RequireAuth(http.HandlerFunc(handlers.MeHandler)).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("me without auth: expected 401, got %d", rr.Code)
	}

	// 8) /me with expired token → 401.
	expClaims := &Claims{
		UserID:   meResp.ID,
		Username: meResp.Username,
		Role:     meResp.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, expClaims).SignedString(jwtSecret)
	req = httptest.NewRequest("GET", "/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rr = httptest.NewRecorder()
	RequireAuth(http.HandlerFunc(handlers.MeHandler)).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("me expired token: expected 401, got %d", rr.Code)
	}
}

func TestIntegrationSeedAdminIdempotent(t *testing.T) {
	db := setupIntegrationDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer db.Exec("DELETE FROM users WHERE username = $1", "seed_test_admin")

	store := NewUserStore(db)
	handlers := NewAuthHandlers(store)

	handlers.SeedAdmin("seed_test_admin", "admin123")
	handlers.SeedAdmin("seed_test_admin", "admin123")

	user, err := store.GetUserByUsername("seed_test_admin")
	if err != nil {
		t.Fatalf("seeded admin not found: %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("expected role 'admin', got %q", user.Role)
	}
	if !CheckPassword(user.PasswordHash, "admin123") {
		t.Error("seeded admin password does not match")
	}
}

func TestIntegrationPasswordIsBcryptHashed(t *testing.T) {
	db := setupIntegrationDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer db.Exec("DELETE FROM users WHERE username = $1", "hash_check_user")

	store := NewUserStore(db)
	if err := store.CreateUser("hash_check_user", "mypassword", "viewer"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := store.GetUserByUsername("hash_check_user")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	if user.PasswordHash == "mypassword" {
		t.Error("password stored as plaintext — must be bcrypt-hashed")
	}
	if user.PasswordHash == "" {
		t.Error("password hash is empty")
	}
	if len(user.PasswordHash) < 20 {
		t.Errorf("password hash suspiciously short: %d chars", len(user.PasswordHash))
	}
}
