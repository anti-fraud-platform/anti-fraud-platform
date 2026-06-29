# Anti-Fraud Platform

<img src="docs/anti-fraud.png" width="150" />

Real-time click fraud detection for ad traffic. Built with Go, Redis, PostgreSQL and React.

Each click hits three checks: a Bloom filter against 12,000 known bad IPs, a Redis rate limiter (5 req/s per IP), and an async batch logger that writes to Postgres every 500ms. A separate analytics service reads from the same DB and powers the dashboard.

![Infra](docs/infra.png)

## Stack

| Service | Port | What it does |
|---|---|---|
| engine | 8080 | Accepts clicks, runs fraud checks |
| analytics | 8081 | REST API over click_logs |
| frontend | 3001 | React dashboard, polls every 2.5s |
| postgres | 5433 | Stores all click logs |
| redis | 6380 | Per-IP rate limit counters |
## Database
PostgreSQL is the only persistent store. The schema is in `deployments/init-db.sql` and runs automatically on first `docker compose up`.

Two tables:

`click_logs` stores every click that hits the engine, both allowed and blocked. The `reason` column holds one of three values: `allowed`, `rate_limit_exceeded`, or `static_blacklist`. Indexes on `ip`, `campaign_id`, and `processed_at` keep the analytics queries fast even at high row counts.

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
WHERE reason = 'static_blacklist'
GROUP BY ip ORDER BY hits DESC LIMIT 10;
```
![Schema](docs/Databaseschema.jpeg)

To wipe all data and start fresh:

```bash
docker compose down -v
docker compose up --build
```
## Getting started

You need Go 1.26+, Docker with Compose, and Git.

```bash
git clone git@github.com:kage-ops-dev/anti-fraud-platform.git
cd anti-fraud-platform
docker compose up --build
```

Once all containers are healthy:

- Dashboard: http://localhost:3001
- Engine: http://localhost:8080/v1/click
- Analytics: http://localhost:8081/v1/analytics/stats

Full setup guide with troubleshooting: [docs/SETUP.md](docs/SETUP.md)

## Sending a click

```bash
curl -X POST http://localhost:8080/v1/click \
  -H "Content-Type: application/json" \
  -d '{"campaign_id":"demo","user_agent":"curl/test","timestamp":1234567890}'
```

| Response | Reason |
|---|---|
| 200 | Click accepted |
| 403 | IP is on the static blacklist |
| 429 | More than 5 requests per second from this IP |
![How It works](docs/detection%20pipeline.jpeg)

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
ok   anti-fraud/internal/bloom    1.4s
ok   anti-fraud/internal/engine   1.5s
```

Notable tests:

- `TestHandleClickIgnoresSpoofedBodyIP` - confirms the body `ip` field is ignored, rate limiting always uses the real connection IP
- `TestHandleClickSelfHealsKeyMissingTTL` - confirms a Redis key that lost its TTL recovers automatically via ExpireNX on the next request
- `TestIPFilter_LogicAndMemory` - Bloom filter loads 12,000 IPs with 0 bytes of unexpected memory growth
- `TestClickIntegrationPipeline` - full HTTP round trip against a real Redis instance
![Sustained load test result](docs/Sustained%20load%20test%20result.jpeg)
## Analytics API

### GET /v1/analytics/stats

```json
{
  "total_clicks": 315936,
  "allowed_count": 15679,
  "blocked_count": 300257,
  "budget_saved": 1501285,
  "top_blocked_ips": [
    { "ip": "1.2.3.4", "count": 28154 }
  ],
  "campaigns": [
    {
      "campaign_id": "camp_alpha_001",
      "total_clicks": 52341,
      "blocked_bots": 48120,
      "saved_money_usd": 240600
    }
  ]
}
```

### GET /v1/analytics/logs

Paginated click log. Query params: `page`, `limit`, `campaign_id`, `is_bot`, `reason`, `from`, `to`.

`from` and `to` accept RFC3339 or `YYYY-MM-DD` format.

### GET /v1/analytics/blacklist/ips

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
  "static_blacklist": 43,
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

![Architecture](docs/Architecture.jpeg)

## CI

GitHub Actions runs on every push to main and every pull request targeting main.

The backend job spins up Redis 7, then runs:

```bash
go build ./...
go test $(go list ./... | grep -v frontend) -race -count=1
```

See [.github/workflows/ci.yml](.github/workflows/ci.yml).
![CI/CD](docs/CICD.jpeg)

## Conclusion

The platform was built and verified end to end over several weeks as a team. The engine handles sustained attack traffic at 1000 rps with a 99.5% catch rate and flat memory at around 36 MB over 10 minute runs. Two real bugs were found and fixed during testing: a rate limiter that trusted the client-supplied IP in the request body (letting bots spoof their way past it), and a Redis TTL race condition that could permanently block an IP even after the attack stopped. Both have regression tests that will catch them if they come back. CI runs on every push to main and covers build, race detection, and all 9 tests including a full HTTP integration test against real Redis.