# Task 9: unit tests for bloom filter

Now test is in anti-fraud-platform/internal/bloom/bloom_test.go
i checked, test passed

there are 2 checks:
1. logic, bad ip is identified as bad, good as good
2. memory sanity, so it verifies that there are no memory leaks or inefficient memory allocations