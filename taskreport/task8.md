## Task 8 Report: Rate Limiter Unit Tests
### 1. What was implemented
I refactored the fixed-window rate limiter and decoupled the time dependency using a custom `Clock` interface to allow fast and deterministic unit testing.
### 2. Conclusion
The unit tests successfully verify all threshold boundaries and time window resets in 0.00 seconds without using slow time.Sleep calls.