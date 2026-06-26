# Task 2 — Engine Stability and Rate Limiter Report
## Scope

This report documents Task 2: stabilizing `cmd/engine`, confirming that the Redis-backed rate limiter works correctly, and recording the commands/results used for final reporting.

The goal was to confirm that `cmd/engine` compiles, runs locally, accepts real click traffic through `POST /v1/click`, applies rate limiting correctly, writes logs asynchronously to PostgreSQL, and remains stable under sustained generator load.

## What Was Completed

- `cmd/engine` builds successfully.
- Redis-backed per-IP rate limiting was confirmed.
- `POST /v1/click` accepts JSON payloads with:
  - `ip`
  - `user_agent`
  - `campaign_id`
  - `timestamp`
- Normal generator traffic returns mostly `200 OK`.
- Attack generator traffic returns `429 Too Many Requests`.
- Click logs are flushed asynchronously to PostgreSQL through `BatchLogger`.
- Integration tests were added for `/v1/click`.
- Sustained 10-minute generator load test completed without crashes or errors.
- Engine memory usage stayed stable during the load test.

## Commands Used

### Start Dependencies

```bash
docker compose up -d postgres redis
```

### Run Engine Locally

```bash
DB_PORT=5433 REDIS_PORT=6380 go run ./cmd/engine
```

### Build

```bash
go build ./...
```

### Run Tests

```bash
go test ./...
```

### Manual JSON Request

```bash
curl -i -X POST http://localhost:8080/v1/click \
  -H "Content-Type: application/json" \
  -d '{"ip":"9.9.9.9","user_agent":"test-agent","campaign_id":"camp_test","timestamp":123456789}'
```

Expected result: `200 OK`.

### Normal Generator Test

```bash
go run ./cmd/generator/ -workers 10 -rps 10 -duration 30s
```

Observed result:

```text
Clean Clicks (200): 2970
Rate-Limit Hits(429): 0
Blacklist Hits (403): 10
Errors (other): 0
```

### Attack Generator Test

```bash
go run ./cmd/generator/ -attack -workers 10 -duration 30s
```

Observed result:

```text
Clean Clicks (200): 150
Rate-Limit Hits(429): 28132
Blacklist Hits (403): 0
Errors (other): 0
Overall Catch Rate: 99.5%
```

### Sustained Load Test

```bash
go run ./cmd/generator/ -workers 20 -rps 50 -duration 10m
```

Observed result:

```text
Total Requests Sent: 575713
Clean Clicks (200): 573758
Rate-Limit Hits(429): 0
Blacklist Hits (403): 1955
Errors (other): 0
Duration: 600.0s
```

### Memory Check

```bash
ps -o pid,rss,vsz,etime -p $(pgrep -f "cmd/engine|/engine")
```

Observed memory readings stayed stable during the 10-minute load test:

```text
00:33 -> RSS 35328 / 24528
05:23 -> RSS 28640 / 24784
10:00 -> RSS 28624 / 25504
10:35 -> RSS 28624 / 25536
```

### Conclusion

Task 2 is completed locally. The engine builds, runs with Redis and PostgreSQL, accepts the required click JSON format, returns `200 OK` for normal traffic, returns `429 Too Many Requests` under attack traffic, writes logs to PostgreSQL, and runs continuously under generator load without crashes or visible memory leaks.

### The remaining verification step is to repeat the same checks on the team VM.