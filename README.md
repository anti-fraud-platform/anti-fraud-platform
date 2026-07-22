# Anti-Fraud Platform

<img src="docs/anti-fraud.png" width="150" />

Real-time click fraud detection for ad traffic. Built with Go, Redis, PostgreSQL and React.

Each click goes through four checks: automated user-agent detection, a GeoIP / ASN policy based on MaxMind `.mmdb` databases, a Redis rate limiter (5 req/s per IP), and an async batch logger that writes to Postgres every 500ms. A separate analytics service reads from the same DB and powers the dashboard.

![Infra](docs/infra.png)

## Stack

| Service | Port | What it does |
|---|---|---|
| engine | 8080 (internal only) | Accepts clicks, runs fraud checks |
| nginx_engine | 9090 | Reverse proxy in front of the engine, serves the click simulator page |
| analytics | 8081 | REST API over click_logs + JWT auth |
| frontend | 3001 | React dashboard, polls every 2.5s |
| postgres | 5433 | Stores all click logs |
| redis | 6380 | Per-IP rate limit counters |

The engine no longer exposes a port directly to the host. All click traffic goes through nginx on port 9090.

Monitoring and VM load-test flow for Week 6: [docs/MONITORING_LOADTEST.md](docs/MONITORING_LOADTEST.md)

## Database

PostgreSQL is the only persistent store. The schema is managed by a migration system (`internal/migrator/`) and applied automatically on first `docker compose up` or when either the engine or analytics service starts.

Five tables:

`click_logs` stores every click that hits the engine, both allowed and blocked. The `reason` column can hold `allowed`, `dynamic_blacklist`, `geoip_policy`, `rate_limit_exceeded`, `suspicious_agent`, `no_js_challenge`, `challenge_too_fast`, `challenge_mismatch`, `suspicious_headers`, or `risk_score_exceeded`. Indexes on `ip`, `campaign_id`, and `processed_at` keep the analytics queries fast even at high row counts.

`campaigns` stores per-campaign configuration including `cost_per_click`. Seeded with `unknown` and `demo` campaigns on first run. Cost per click can be updated via `PUT /v1/analytics/campaigns` or edited inline in the dashboard.

`dynamic_blacklist` stores IPs auto-promoted from repeated flagged clicks (5+ hits within 1 hour). Entries expire after 15 minutes. Used for fast blocking at the engine layer.

`users` stores authenticated user accounts with bcrypt-hashed passwords and roles (`admin`, `viewer`). An admin user is auto-seeded on first startup when `REQUIRE_AUTH=true`.

`audit_events` stores system events for the activity feed. Empty by default, populated manually or via an application hook.

Connect directly to inspect:

```bash
docker exec -it antifraud-postgres psql -U antifraud -d analytics
```

Useful queries:

```sql
-- total rows
SELECT count(*) FROM click_logs;

-- breakdown by reason
SELECT reason, count(*) FROM click_logs GROUP BY reason;

-- top blocked IPs
SELECT ip, count(*) as hits FROM click_logs
WHERE reason = 'geoip_policy'
GROUP BY ip ORDER BY hits DESC LIMIT 10;
```

![Schema](docs/Databaseschema.jpeg)

To wipe all data and start fresh:

```bash
docker compose down -v
docker compose up --build
```

## Getting started

### Prerequisites

You need Go 1.26+, Docker with Compose support, and Git.

```bash
go version
docker --version
docker compose version
git --version
```

### 1. Clone and build

```bash
git clone git@github.com:anti-fraud-platform/anti-fraud-platform.git
cd anti-fraud-platform
docker compose up --build -d
```

If SSH access to GitHub is not configured on the machine, use HTTPS instead:

```bash
git clone https://github.com/anti-fraud-platform/anti-fraud-platform.git
cd anti-fraud-platform
```

This builds and starts six containers: `engine`, `nginx_engine`, `analytics`, `frontend`, `postgres`, `redis`. The first build takes 1-3 minutes depending on your machine; subsequent runs are faster since Docker caches layers.

The repository already includes the MaxMind databases at `geoip/GeoLite2-Country.mmdb`, `geoip/GeoLite2-City.mmdb`, and `geoip/GeoLite2-ASN.mmdb`. The engine image copies them at build time, so GeoIP / ASN rules work out of the box in local Docker and Railway.

If your local Postgres volume already existed from an older project week, the services now apply the current schema automatically on startup. You do not need to wipe the volume just to pick up new tables like `campaigns` or new columns like `risk_reasons`.

### 2. Confirm everything is healthy
Check the containers:

```bash
docker compose ps
```

Expected output, all six services `Up` (postgres and redis should additionally say `(healthy)`):

```
NAME                     SERVICE        STATUS
antifraud-analytics      analytics      Up
antifraud-engine         engine         Up
antifraud-frontend       frontend       Up
antifraud-nginx-engine   nginx_engine   Up
antifraud-postgres       postgres       Up (healthy)
antifraud-redis          redis          Up (healthy)
```

Note that `antifraud-engine` has no host port mapped (only `8080/tcp` internal). That's correct, the engine is intentionally not reachable directly. All click traffic goes through `nginx_engine` on port 9090.

If any container shows `Restarting` or `Exited`, check its logs:

```bash
docker compose logs <service-name>
```

### 3. Access the UI

| What | URL |
|---|---|
| Dashboard | http://localhost:3001 |
| Login | http://localhost:3001/login |
| Register | http://localhost:3001/register |
| Click simulator | http://localhost:9090 |

Open the dashboard at http://localhost:3001. With auth enabled (`REQUIRE_AUTH=true`), you will be redirected to `/login`. Log in with `admin / admin123` (or register a new account). Once authenticated, you should see four stat cards (Total clicks, Blocked clicks, Allowed clicks, Active campaigns) with delta percentages, updating automatically every 2.5 seconds via polling.

Open the click simulator in a separate tab and click the buttons a few times. `Send Real Click` solves the JS challenge in-browser and should produce a `success` response. `Send Click Without Solving Challenge` should produce a `flagged` response. Within a couple seconds, refresh the dashboard and confirm the numbers moved.

### 4. Verify via terminal

Generate some click traffic through nginx and check it landed in the database:

```bash
# challenge should exist and return challenge_id + nonce
curl -s http://localhost:9090/v1/challenge | python3 -m json.tool
```

```bash
# unsolved click should be flagged, not silently accepted
curl -s -X POST http://localhost:9090/click -H "Content-Type: application/json" -d '{"campaign_id":"demo"}'
```

Expected response:

```json
{"status":"flagged","message":"Click accepted for validation analysis pipeline"}
```

That result is expected. A raw curl request does not solve the JS challenge, so the engine should mark it as suspicious. For a real `success` path, use the simulator page at `http://localhost:9090` and click `Send Real Click`, which fetches `/v1/challenge`, solves it in the browser, waits briefly, and then submits the click.

```bash
# simulated bot click
curl -s -X POST http://localhost:9090/bot/click -H "Content-Type: application/json"
```

Expected response:

```json
{"status":"flagged","message":"Click accepted for validation analysis pipeline"}
```

Confirm the engine itself is not reachable directly (this should fail to connect, not return an error JSON):

```bash
curl -i http://localhost:8080/v1/click
```

Expected:

```
curl: (7) Failed to connect to localhost port 8080
```

Check the rows landed in Postgres:

```bash
docker exec -it antifraud-postgres psql -U antifraud -d analytics -c \
  "SELECT ip, reason, is_bot, user_agent FROM click_logs ORDER BY processed_at DESC LIMIT 5;"
```

Expected: at least one recent row with `is_bot = t`. A raw curl request usually lands as `reason = risk_score_exceeded` or `reason = no_js_challenge`, depending on the headers you sent. A simulator-driven real click should land as `reason = allowed`.

Confirm the analytics API picked it up:

```bash
curl -s http://localhost:8082/v1/analytics/stats | python3 -m json.tool
```

`total_clicks` should now be at least 2, and `blocked_count` should reflect the bot click.

### 5. Run the test suite

```bash
make ci-backend
```

Expected: all packages report `ok`, no `FAIL`. See the [Tests](#tests) section below for what each test proves.

To run the frontend validation that CI uses:

```bash
make ci-frontend
```

To run the full Docker smoke test locally the same way CI does:

```bash
make ci-compose-up
make ci-compose-smoke
make ci-compose-down
```

### 6. Start monitoring (optional but recommended)

```bash
COMPOSE_PROFILES=monitoring docker compose up --build -d
```

Open:

- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9091`

Grafana login defaults to `admin / admin`.

The provisioned dashboard already shows:

- request rate on `/v1/click`
- `200 / 403 / 429` over time
- `p95` click latency
- engine goroutine count
- Redis / PostgreSQL health
- node CPU / memory

### 7. Run the Week 6 load tests

Real-click ramp:

```bash
bash scripts/loadtest/run_k6.sh k6_real_click_ramp.js
```

Mixed screenshot run:

```bash
bash scripts/loadtest/run_k6.sh k6_status_mix.js
```

If you are targeting the university VM instead of local Docker, point both commands at the VM gateway:

```bash
BASE_URL=http://10.93.26.161:9090 bash scripts/loadtest/run_k6.sh k6_real_click_ramp.js
```

### Tearing down

```bash
docker compose down        # stop containers, keep data
docker compose down -v     # stop containers, wipe Postgres volume too
```

Full setup guide with additional troubleshooting: [docs/SETUP.md](docs/SETUP.md)

## Authentication

The analytics API supports JWT-based authentication. When enabled, all `/v1/analytics/*` endpoints require a valid Bearer token.

### Enabling auth

Set these environment variables in `docker-compose.yml`:

```yaml
analytics:
  environment:
    REQUIRE_AUTH: "true"
    JWT_SECRET: "your-secret-key"
    ADMIN_PASSWORD: "your-admin-password"
```

On first startup, an admin user (`admin`) is auto-seeded. All users registered via the API default to the `viewer` role.

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

### Disabling auth

Set `REQUIRE_AUTH` to `false` or leave it unset. All endpoints will be open (this is the default for local development and CI).

## Configuration

Key environment variables for the stack:

| Variable | Service | Default | Description |
|---|---|---|---|
| `REQUIRE_AUTH` | analytics | `false` | Enable JWT auth on analytics endpoints |
| `JWT_SECRET` | analytics | `change-me-in-production` | Secret key for signing JWT tokens |
| `ADMIN_PASSWORD` | analytics | `admin123` | Password for the auto-seeded admin user |
| `REQUIRE_JS_CHALLENGE` | engine | `true` | Enable JS challenge verification |
| `REQUIRE_HEADER_CHECK` | engine | `true` | Enable header heuristic scoring |
| `DB_HOST` | all | `localhost` | PostgreSQL host |
| `DB_PORT` | all | `5432` | PostgreSQL port |
| `DB_USER` | all | `antifraud` | PostgreSQL username |
| `DB_PASSWORD` | all | `antifraud123` | PostgreSQL password |
| `DB_NAME` | all | `analytics` | PostgreSQL database name |

All variables are set in `docker-compose.yml` and `docker-compose.ci.yml`. For the full list, see the compose files.

## Real GeoIP Checks

GeoIP only makes sense if the databases are real and the IP is a real public address.

The repository already contains `GeoLite2-Country.mmdb`, `GeoLite2-City.mmdb`, and `GeoLite2-ASN.mmdb`, so you can verify the direct lookup locally right away with:

```bash
go run ./cmd/geoiplookup -ip 8.8.8.8
```

That command reads all three MaxMind databases the engine uses at runtime.

For the full manual e2e path through nginx, engine, batch logging, and Postgres, run:

```bash
bash scripts/geoip/e2e_real_ip.sh
```

## University VM Deployment

The current hosted environment runs on the university VM `afplatform` with Ubuntu 22.04.
The stack is deployed with Docker Compose and is reachable from the Innopolis University internal network or through the university VPN.

Current endpoints on the VM:

| What | URL |
|---|---|
| Dashboard | `http://10.93.26.161:3001` |
| Analytics API | `http://10.93.26.161:8082/v1/analytics/stats` |
| Engine simulator | `http://10.93.26.161:9090` |

The engine itself is not exposed directly. The VM publishes the simulator page on port `9090`, which proxies requests to the internal `engine` service.

To verify the deployed VM stack from inside the university network:

```bash
curl http://10.93.26.161:8082/v1/analytics/stats
curl -X POST http://10.93.26.161:9090/click -H "Content-Type: application/json" -d '{}'
curl -X POST http://10.93.26.161:9090/bot/click -H "Content-Type: application/json" -d '{}'
```

## Sending a click

The engine is not reachable directly. All clicks go through nginx on port 9090.

```bash
curl -X POST http://localhost:9090/click \
  -H "Content-Type: application/json" \
  -d '{"campaign_id":"demo"}'
```

| Response | Reason |
|---|---|
| 200, status: success | Challenge was solved and the request looked like a normal browser click |
| 200, status: flagged | The click was accepted for logging, but one of the fraud checks marked it suspicious |
| 403 | IP is on the static blacklist |
| 429 | More than 5 requests per second from this IP |

![How It works](docs/detection%20pipeline.jpeg)

## Click simulator

A small page served by nginx at `http://localhost:9090` for testing the detection logic without writing curl commands. It has three buttons:

`Send Real Click` fetches `/v1/challenge`, solves it in the browser, waits briefly, and then sends the click.

`Send Click Without Solving Challenge` uses the same browser session but skips the challenge step, so it should come back as `flagged`.

`Legacy Naive Bot Flag (before)` sends the old forced bot-style request through `/bot/click`. It is kept for comparison with the newer challenge-based flow.

The important part is the layered behavior. A suspicious user-agent is still caught, but the main proof now is that a browser-shaped request without a solved JS challenge does not pass as a clean click.

## Traffic generator

```bash
# Normal traffic, 10 workers at 10 rps for 30 seconds
go run ./cmd/generator/ -workers 10 -rps 10 -duration 30s

# Attack simulation, one IP hammering at 1000 rps
go run ./cmd/generator/ -attack -workers 10 -duration 30s
```

Normal mode output from our tests:

```
Total Requests Sent : 2980
Clean Clicks (200)  : 2968
Rate-Limit Hits(429): 0
Blacklist Hits (403): 12
```

Attack mode output:

```
Total Requests Sent : 28412
Clean Clicks (200)  : 150
Rate-Limit Hits(429): 28262
Overall Catch Rate  : 99.5%
```

## Tests

```bash
go test $(go list ./... | grep -v frontend) -race -count=1
```

```
ok   anti-fraud/cmd/engine        1.8s
ok   anti-fraud/internal/geopolicy 1.4s
ok   anti-fraud/internal/engine   1.5s
```

Notable tests:

- `TestHandleClickIgnoresSpoofedBodyIP` - confirms the body `ip` field is ignored, rate limiting always uses the real connection IP
- `TestHandleClickSelfHealsKeyMissingTTL` - confirms a Redis key that lost its TTL recovers automatically via ExpireNX on the next request
- `TestHandleClickSuspiciousAgentDetection` - confirms bot user-agents (curl, python-requests, empty UA, explicit automated header) are all flagged correctly
- `TestEvaluateMatchesBlockedASNKeyword` - confirms GeoIP / ASN policy blocks configured network organizations
- `TestClickIntegrationPipeline` - full HTTP round trip against a real Redis instance
- `TestValidateTokenHS384RejectedWhenExpectingHS256` - confirms algorithm confusion attacks are blocked (auth)
- `TestIntegrationRegisterLoginMeFlow` - full register → login → me flow against real Postgres (auth)
- `TestIntegrationSeedAdminIdempotent` - confirms admin user is seeded correctly on startup (auth)

![Sustained load test result](docs/Sustained%20load%20test%20result.jpeg)

## Analytics API

All endpoints below are protected when `REQUIRE_AUTH=true`. Pass `Authorization: Bearer <token>` in the request header.

Full API contract: [docs/api.md](docs/api.md)

### POST /v1/auth/register

Create a new user account.

```json
{ "username": "alice", "password": "pass123" }
```

Response: `201 { "message": "user created" }`

Validates: username 3-64 chars, password 6+ chars. Returns `409` on duplicate username.

### POST /v1/auth/login

Authenticate and receive a JWT token.

```json
{ "username": "alice", "password": "pass123" }
```

Response: `200 { "token": "eyJhbGci..." }`

Returns `401` on invalid credentials. Token expires in 24 hours.

### GET /v1/auth/me

Return the current user's profile. Requires auth.

```json
{ "id": 1, "username": "alice", "role": "viewer" }
```

### GET /v1/analytics/stats

```json
{
  "total_clicks": 315936,
  "allowed_count": 15679,
  "blocked_count": 300257,
  "blocked_bots": 300257,
  "budget_saved": 1501285,
  "saved_money_usd": 1501285,
  "top_blocked_ips": [
    { "ip": "1.2.3.4", "blocked": 28154, "total_requests": 29000 }
  ],
  "campaigns": [
    {
      "campaign_id": "camp_alpha_001",
      "total_clicks": 52341,
      "blocked_bots": 48120,
      "saved_money_usd": 240600
    }
  ],
  "previous_total_clicks": 290000,
  "previous_blocked_count": 250000,
  "total_clicks_delta_percent": 8.9,
  "blocked_count_delta_percent": 20.1,
  "reason_breakdown": {
    "suspicious_agent": 5,
    "no_js_challenge": 11,
    "suspicious_headers": 9,
    "geoip_policy": 13,
    "rate_limit_exceeded": 6,
    "risk_score_exceeded": 3
  },
  "js_challenge_blocked": 11,
  "header_heuristic_blocked": 9
}
```

### GET /v1/analytics/logs

Paginated click log. Query params: `page`, `limit`, `campaign_id`, `is_bot`, `reason`, `from`, `to`.

`reason` accepts the stored click reasons such as `allowed`, `dynamic_blacklist`, `geoip_policy`, `rate_limit_exceeded`, `suspicious_agent`, `no_js_challenge`, `challenge_too_fast`, `challenge_mismatch`, `suspicious_headers`, and `risk_score_exceeded`.

`from` and `to` accept RFC3339 or `YYYY-MM-DD` format.

### GET /v1/analytics/blacklist/ips

Only includes IPs blocked via the GeoIP / ASN policy (`reason = geoip_policy`). Clicks flagged by user-agent detection don't appear here, they're visible through `/v1/analytics/logs` filtered by `reason=suspicious_agent`.

```json
{
  "items": [
    {
      "ip": "87.32.171.138",
      "block_count": 3,
      "first_blocked": "2026-06-29 19:36",
      "last_blocked": "2026-06-30 01:12"
    }
  ],
  "total": 12
}
```

### GET /v1/analytics/blacklist/summary

```json
{
  "total_blocked": 300257,
  "geoip_policy_blocked": 43,
  "rate_limited": 300214,
  "auto_blocked_24h": 39
}
```

### GET /v1/analytics/trend

7 day breakdown for the trend chart.

```json
{
  "data": [
    {
      "date": "2026-06-28",
      "total_clicks": 315936,
      "allowed_count": 15679,
      "blocked_count": 300257
    }
  ]
}
```

### GET /v1/analytics/events

Last 20 audit events. Requires rows in the `audit_events` table.

### PUT /v1/analytics/campaigns

Update or create a campaign with a custom cost per click.

```json
{ "campaign_id": "demo", "cost_per_click": 10 }
```

Response: `200 { "campaign_id": "demo", "cost_per_click": 10 }`

Upserts into the `campaigns` table. Validates: `campaign_id` required, `cost_per_click` must be positive. The dashboard's "Cost saved by campaign" widget also supports inline editing of this value.

![Architecture](docs/Architecture.jpeg)

## CI

GitHub Actions runs on every push to `main` and every pull request targeting `main`.

The workflow now has five jobs:

```bash
backend:
  go mod tidy
  git diff --exit-code -- go.mod go.sum
  go run ./cmd/migrate up
  go test ./...
  go vet ./...

compose-config:
  docker compose config
  docker compose --profile monitoring config
  docker compose -f docker-compose.ci.yml config

frontend:
  npm ci
  npm run lint
  npm run build

docker-build:
  docker build -f Dockerfile.engine -t antifraud-engine:ci .
  docker build -f Dockerfile.analytics -t antifraud-analytics:ci .
  docker build -f Dockerfile.nginx-engine -t antifraud-nginx-engine:ci .
  docker build -f frontend/Dockerfile -t antifraud-frontend:ci .

integration:
  docker compose up --build -d
  bash scripts/ci/compose_smoke.sh
```

The frontend build is uploaded as a `frontend-dist` artifact instead of being discarded at the end of the job.

The backend job now uses real PostgreSQL and Redis service containers, applies migrations before tests, and verifies that `go.mod` / `go.sum` stay clean after `go mod tidy`.

The integration stage boots the real production-like stack and checks the behavior that mattered in review:

- the dashboard shell loads on `:3001`
- the click simulator page loads on `:9090`
- `/v1/challenge` returns a real challenge payload
- a click without a solved challenge comes back as `flagged`
- analytics returns the new fields such as `reason_breakdown`, `js_challenge_blocked`, and `header_heuristic_blocked`
- nginx still reaches the engine after recreating only the `engine` container, which catches the stale-upstream bug we hit earlier

On failure, the workflow uploads compose diagnostics so the broken stack can be inspected without rerunning locally.

See [.github/workflows/ci.yml](.github/workflows/ci.yml), [scripts/ci/compose_smoke.sh](scripts/ci/compose_smoke.sh), and [scripts/ci/README.md](scripts/ci/README.md).

![CI/CD](docs/CICD.jpeg)

## Conclusion

The platform was built and verified end to end over several weeks as a team. The engine handles sustained attack traffic at 1000 rps with a 99.5% catch rate and flat memory at around 36 MB over 10 minute runs. Three real bugs were found and fixed during testing: a rate limiter that trusted the client-supplied IP in the request body (letting bots spoof their way past it), a Redis TTL race condition that could permanently block an IP even after the attack stopped, and a missing User-Agent on the manual click route that caused legitimate API testing to be misclassified as bot traffic. All three have regression tests that will catch them if they come back. The engine also now detects automated traffic by user-agent before it reaches the blacklist or rate limiter, independent of any client-supplied headers. CI runs on every push to main and covers build, race detection, and the full test suite including HTTP integration tests against real Redis.
