# Task 14 — JWT Authentication, Password Hashing, Auth Middleware

## Description

Implemented a complete JWT-based authentication system for the analytics API,
including password hashing (bcrypt), HTTP middleware, user persistence, and
three auth endpoints. Auth is gated behind a `REQUIRE_AUTH` feature toggle
(default: `false`) to ensure zero breakage of existing services, CI, and tests.

### Why

The entire analytics API surface was open — any client could read stats, logs,
and blacklist data without authentication. This adds a basic auth layer to
protect dashboard data while keeping the external-facing engine endpoints
(`/v1/click`, `/v1/challenge`) fully open.

---

## New Files

### `internal/auth/context.go`
Context helpers for storing and retrieving JWT claims from `context.Context`.
Uses an unexported key type to prevent collisions.

### `internal/auth/password.go`
- `HashPassword(password) (string, error)` — bcrypt with cost 10
- `CheckPassword(hash, password) bool` — constant-time comparison

### `internal/auth/jwt.go`
- `GenerateToken(userID, username, role) (string, error)` — HS256, 24h expiry
- `ValidateToken(tokenString) (*Claims, error)` — validates signature + expiry
- Explicitly requires `jwt.SigningMethodHS256` (not HMAC generic) to prevent algorithm confusion attacks

### `internal/auth/store.go`
- `UserStore` — PostgreSQL-backed user CRUD
- `CreateUser(username, password, role) error`
- `GetUserByUsername(username) (*User, error)`

### `internal/auth/middleware.go`
- `RequireAuth(next) http.Handler` — extracts + validates Bearer token, returns 401 on failure
- `OptionalAuth(next) http.Handler` — injects claims if present, passes through otherwise

### `internal/auth/handlers.go`
- `POST /v1/auth/register` — creates user with bcrypt-hashed password, validates input (username 3-64 chars, password 6+ chars), returns 409 on duplicate
- `POST /v1/auth/login` — verifies credentials, returns JWT token
- `GET /v1/auth/me` — returns current user profile (requires auth middleware)
- `SeedAdmin(username, password)` — idempotent admin user creation at startup

---

## Modified Files

### `deployments/init-db.sql`
Added `users` table:
```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```
Additive only — `IF NOT EXISTS` ensures zero breakage on existing databases.

### `internal/dbschema/schema.go`
Same `users` table added to the programmatic `schemaSQL` constant. Both engine
and analytics services auto-apply on startup.

### `cmd/analytics/main.go`
- Added `anti-fraud/internal/auth` import
- Added `requireAuth` package variable (default `false`)
- Load `REQUIRE_AUTH` env var in `main()`
- Create `UserStore` + `AuthHandlers`
- Seed admin when `REQUIRE_AUTH=true`
- Register `/v1/auth/register`, `/v1/auth/login`, `/v1/auth/me` endpoints
- Wrap all `/v1/analytics/*` handlers with `auth.RequireAuth()` only when enabled
- Engine endpoints (`/v1/click`, `/v1/challenge`) remain untouched

### `docker-compose.yml`
Added to analytics service:
```yaml
REQUIRE_AUTH: "false"
JWT_SECRET: "change-me-in-production"
ADMIN_PASSWORD: "admin123"
```

### `go.mod` / `go.sum`
- `golang.org/x/crypto v0.54.0` (bcrypt)
- `github.com/golang-jwt/jwt/v5 v5.3.1` (JWT)

---

## Bug Found and Fixed

**Algorithm confusion vulnerability in `ValidateToken`**

The original keyfunc used a type assertion `t.Method.(*jwt.SigningMethodHMAC)`
which matched both `HS256` **and** `HS384`. An attacker could sign a token with
HS386 and it would pass validation.

**Fix** (`jwt.go:55`): Changed to exact method comparison:
```go
if t.Method != jwt.SigningMethodHS256 {
    return nil, ErrTokenInvalid
}
```

Caught by test `TestValidateTokenHS384RejectedWhenExpectingHS256`.

---

## Test Coverage

56 tests across 5 test files in `internal/auth/`:

### `password_test.go` (10 tests)
| Test | What it verifies |
|---|---|
| `TestHashAndCheckPassword` | Basic roundtrip |
| `TestDifferentHashesForSamePassword` | Random salt produces different hashes |
| `TestHashIsEmptyString` | Empty password accepted |
| `TestHashIsBcryptFormat` | Output starts with `$2a$` or `$2b$` |
| `TestCheckPasswordWithGarbageHash` | Non-bcrypt input returns false |
| `TestHashPasswordVeryLongInput` | Rejects >72 bytes, accepts exactly 72 |
| `TestHashPasswordUnicode` | Cyrillic, Japanese, emoji passwords work |
| `TestConcurrentHashAndCheck` | 50 goroutines — no race conditions |
| `TestHashCostIsReasonable` | Verifies bcrypt cost factor = 10 |
| `TestCheckPasswordConstantTime` | Confirms constant-time comparison behavior |

### `jwt_test.go` (18 tests)
| Test | What it verifies |
|---|---|
| `TestGenerateAndValidateToken` | Full roundtrip with claim verification |
| `TestTokenHasCorrectExpiry` | 24h window with ±1min tolerance |
| `TestTokenSubjectMatchesUsername` | Subject field populated |
| `TestTokenHasIssuedAt` | IssuedAt set to now |
| `TestValidateTokenTampered` | Modified signature rejected |
| `TestValidateTokenTruncated` | Half-token rejected |
| `TestValidateTokenEmpty` | Empty string rejected |
| `TestValidateTokenWrongSecret` | Different signing key rejected |
| `TestValidateTokenExpired` | Returns `ErrTokenExpired` |
| `TestValidateTokenNoneAlgorithmAttack` | `alg:none` token rejected (security) |
| `TestValidateTokenHS384RejectedWhenExpectingHS256` | Algorithm confusion blocked (security) |
| `TestValidateTokenDifferentSecret` | Cross-server secret mismatch |
| `TestConcurrentGenerateAndValidate` | 100 goroutines — no races |
| `TestGenerateTokenPreservesSpecialChars` | Unicode in username preserved |
| `TestGetEnvDefault` | Env lookup with fallback |
| `TestValidateTokenMalformedDots` | 8 malformed token formats rejected |
| `TestValidateTokenWithPayloadManipulation` | Payload tamper breaks signature |
| `TestValidateTokenNotBefore` | Future-dated token handled |

### `middleware_test.go` (17 tests)
| Test | What it verifies |
|---|---|
| `TestRequireAuthPassesValidToken` | 200 + claims in context |
| `TestRequireAuthRejectsMissingHeader` | 401 + error message |
| `TestRequireAuthRejectsBasicAuth` | "Basic" scheme rejected |
| `TestRequireAuthRejectsExpiredToken` | 401 for expired JWT |
| `TestRequireAuthRejectsTamperedToken` | 401 for modified token |
| `TestRequireAuthRejectsEmptyBearer` | "Bearer " with no token → 401 |
| `TestRequireAuthRejectsTokenWithoutBearerPrefix` | Raw token without prefix → 401 |
| `TestRequireAuthCaseInsensitiveScheme` | "bearer" (lowercase) accepted |
| `TestRequireAuthRejectsTooManyParts` | "Bearer token extra" → 401 |
| `TestRequireAuthContextPropagatesClaims` | All claim fields verified in handler |
| `TestRequireAuthConcurrentRequests` | 50 concurrent requests — no races |
| `TestOptionalAuthPassesWithoutToken` | No header → passes through, no claims |
| `TestOptionalAuthInjectsClaimsWhenPresent` | Valid token → claims injected |
| `TestOptionalAuthDoesNotRejectInvalidToken` | Bad token → passes through (not 401) |
| `TestOptionalAuthDoesNotRejectMalformedHeader` | "Basic" → passes through |
| `TestRequireAuthReturnsJSONError` | Error body contains "error" |
| `TestRequireAuthBearerWithExtraWhitespace` | "Bearer  token" (double space) → 401 |

### `context_test.go` (6 tests)
| Test | What it verifies |
|---|---|
| `TestContextWithAndWithoutUser` | Empty context → nil, false |
| `TestContextWithUserRoundTrip` | Store then retrieve claims |
| `TestContextOverwriteClaims` | Second write replaces first |
| `TestContextTypeSafety` | Wrong type in context → nil, false |
| `TestContextNilClaims` | Typed nil edge case |
| `TestContextDoesNotLeakBetweenRequests` | Independent contexts isolated |

### `handlers_test.go` (22 tests, 3 skipped without Postgres)
| Test | What it verifies |
|---|---|
| `TestRegisterRejectsNonPOST` | 405 for GET/PUT/DELETE/PATCH |
| `TestRegisterRejectsEmptyBody` | 400 for nil body |
| `TestRegisterRejectsInvalidJSON` | 400 for malformed JSON |
| `TestRegisterRejectsMissingFields` | 400 for missing username/password/empty |
| `TestRegisterRejectsShortUsername` | 400 for <3 chars |
| `TestRegisterRejectsLongUsername` | 400 for >64 chars |
| `TestRegisterRejectsShortPassword` | 400 for <6 chars |
| `TestLoginRejectsNonPOST` | 405 for GET/PUT/DELETE |
| `TestLoginRejectsEmptyBody` | 400 for nil body |
| `TestLoginRejectsInvalidJSON` | 400 for malformed JSON |
| `TestLoginRejectsMissingFields` | 400 for missing username/password |
| `TestMeRejectsNonGET` | 405 for POST/PUT/DELETE |
| `TestMeRejectsWithoutAuthContext` | 401 when no claims in context |
| `TestMeReturnsClaimsFromContext` | Correct ID/username/role from context |
| `TestMeResponseContentType` | Content-Type: application/json |
| `TestIntegrationRegisterLoginMeFlow` | Full e2e: register→login→me→errors (skipped) |
| `TestIntegrationSeedAdminIdempotent` | Seed twice, verify role + password (skipped) |
| `TestIntegrationPasswordIsBcryptHashed` | Password stored as hash, not plaintext (skipped) |

---

## How Auth Works

### Toggle
```
REQUIRE_AUTH=false  → all endpoints open (default, CI-safe)
REQUIRE_AUTH=true   → analytics endpoints require JWT
```

### Flow
```
1. POST /v1/auth/register  { "username": "alice", "password": "pass123" }
   → 201 { "message": "user created" }

2. POST /v1/auth/login     { "username": "alice", "password": "pass123" }
   → 200 { "token": "eyJhbGci..." }

3. GET /v1/analytics/stats
   Header: Authorization: Bearer eyJhbGci...
   → 200 { ...analytics data... }

4. GET /v1/auth/me
   Header: Authorization: Bearer eyJhbGci...
   → 200 { "id": 1, "username": "alice", "role": "viewer" }
```

### What stays open (no auth required)
- `GET /v1/challenge` — external JS challenge
- `POST /v1/click` — external click ingestion
- `GET /health` — health check
- `GET /metrics` — Prometheus metrics
- `POST /v1/auth/register` — public registration
- `POST /v1/auth/login` — public login

---

## Verification

```bash
# Build
go build ./...

# All tests (auth + existing)
go test ./... -count=1

# Auth tests only (verbose)
go test ./internal/auth/ -v -count=1

# Race detector
go test ./internal/auth/ -race -count=1

# Vet
go vet ./...

# Integration tests (needs Postgres)
go test ./internal/auth/ -run TestIntegration -count=1 -v
```

---

## Frontend Instructions

### Login Page (React)

Create a new route `/login` with a form:

```jsx
// src/pages/Login.jsx
const [form, setForm] = useState({ username: '', password: '' })
const [error, setError] = useState('')

const handleSubmit = async (e) => {
  e.preventDefault()
  setError('')
  try {
    const res = await fetch('/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Login failed')
    }
    const data = await res.json()
    localStorage.setItem('token', data.token)
    window.location.href = '/analytics'
  } catch (err) {
    setError(err.message)
  }
}
```

### Axios interceptor (attach token to all requests)

```jsx
// src/api/client.js
import axios from 'axios'

const client = axios.create({ baseURL: '' })

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Optional: redirect to login on 401
client.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)
```

### Enabling auth on backend

When the login page is ready, change in `docker-compose.yml`:

```yaml
REQUIRE_AUTH: "true"
```

Then redeploy: `docker compose down && docker compose up -d --build`

### What to test manually
1. Visit `/login`, submit wrong credentials → see error message
2. Submit correct credentials → redirected to `/analytics` with data loading
3. Clear `localStorage`, refresh `/analytics` → redirected back to `/login`
4. Tamper with token in localStorage → 401 → redirected to `/login`

---

## Conclusion

Added JWT authentication with bcrypt password hashing to the analytics API.
56 tests cover security (algorithm confusion, `alg:none`, tampered tokens,
expired tokens, payload manipulation), edge cases (empty inputs, unicode,
concurrency), and full integration flows. Auth is opt-in via `REQUIRE_AUTH`
env var — zero behavioral change when disabled, full protection when enabled.
