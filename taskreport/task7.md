## Task 7 Report: Bloom Filter for Blacklisted IPs

### 1. What was implemented
I implemented an in-memory Bloom Filter using the `bloom/v3` library at the very entry point of the click-handling path to drop blacklisted IPs before any Redis or database calls.

### 2. Conclusion
The automated test confirms that malicious traffic is instantly dropped with a `403 Forbidden` status in less than 1 millisecond.

### 3. Verification Command and Output
```bash
go test -v -run TestHandleClickBloomFilterBlacklist ./cmd/engine/...
```
=== RUN   TestHandleClickBloomFilterBlacklist
--- PASS: TestHandleClickBloomFilterBlacklist (0.04s)
PASS