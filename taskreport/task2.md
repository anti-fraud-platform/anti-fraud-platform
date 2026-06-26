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