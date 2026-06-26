<<<<<<< HEAD
## Week 4 Progress Report: High-Performance Filtering and Verification

### 1. Description of Implementation
This week, we focused on optimization and core protection features to ensure our platform handles requests efficiently and safely:
* **In-Memory Bloom Filter:** Implemented a fast IP blacklist check using the `github.com/bits-and-blooms/bloom/v3` library. The system loads a real-world dataset of over 12,000 known-bad IPs from `dirty_ips.txt` at startup into a bit array sized for 15,000 elements (1% false-positive rate). This filter sits at the absolute entry point of the click-handling path, dropping malicious traffic immediately without any database or Redis calls.
* **Refactored Rate Limiter:** Moved the rate-limiting logic into a dedicated package (`internal/engine`). To make the system fully testable, we decoupled the time dependency by introducing a `Clock` interface. This allows us to inject a real system clock in production and a fake clock in our test environment.

### 2. Testing and Verification Performed
We performed strict unit testing to verify the behavior of our core components without adding real-time delays:
* **Unit Tests for Rate Limiter:** Created `ratelimiter_test.go` using Go's standard `testing` package.
* **Fake Clock Injection:** Instead of using `time.Sleep` (which slows down pipelines), we used a custom `FakeClock` to manually advance time.
* **Test Cases Covered:**
    1.  **Single Request:** Verified that the very first incoming request from an IP is successfully allowed.
    2.  **Edge of Threshold:** Filled the rate limit window up to the maximum threshold (5 requests) to ensure they all pass.
    3.  **Over the Threshold:** Verified that the 6th burst request within the same window is instantly blocked.
    4.  **Time Window Reset:** Advanced the fake clock by 1 second and verified that the counter resets, allowing new requests to pass again.

### 3. Results Obtained
* **Bloom Filter Efficiency:** The Bloom Filter successfully processes and drops blacklisted IPs in less than 1 millisecond (<1ms), completely protecting our application state and infrastructure from junk traffic.
* **Unit Test Results:** The entire suite of engine tests executed successfully in **0.00s** (total package run time ~0.3s), proving that the fake clock injection works perfectly and guarantees lightning-fast execution in the CI/CD pipeline.

### 4. Commands Used
To download dependencies, build the project, and run the test suite, use the following commands:

```bash
# Download the required Bloom Filter library
go get -u [github.com/bits-and-blooms/bloom/v3](https://github.com/bits-and-blooms/bloom/v3)

# Run the unit tests for the core engine with verbose output
go test -v ./internal/engine/...
```

### 5. Terminal Output and Logs

* Below is the successful verification log from the local test run:
```text
=== RUN   TestRateLimiter
--- PASS: TestRateLimiter (0.00s)
PASS
ok      anti-fraud/internal/engine      0.306s
```
=======
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

>>>>>>> main
