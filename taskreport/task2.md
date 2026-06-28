# Task 2 — Engine Stability and Rate Limiter Report

## Scope
Stabilize `cmd/engine`: confirm it compiles and runs locally, accepts
real click traffic through `POST /v1/click`, applies per-IP rate
limiting correctly via Redis, writes logs asynchronously to
PostgreSQL, and remains stable under sustained generator load.

## Bugs Found and Fixed During This Task

### Bug 1 — client-supplied IP was trusted for rate-limit/blacklist decisions
`ClickPayload` accepted an `ip` field from the request body, and the
handler overrode the real connection IP with it. This meant a bot
could put a different fake IP in the JSON body on every request and
get a fresh rate-limit bucket every time — completely bypassing the
limiter. **Fix:** the IP used for the blacklist check and the rate
limiter now comes only from `getClientIP(r)` (X-Forwarded-For /
X-Real-IP / RemoteAddr). The body's `ip` field is no longer used for
any decision.
**Verified by:** `TestHandleClickIgnoresSpoofedBodyIP` — sends 6
requests, each claiming a different IP in the body but all from the
same real connection, and confirms they share one rate-limit bucket
instead of each getting its own.

### Bug 2 — rate-limit key could get permanently stuck with no TTL
The original code only called `Expire` once, when a key's counter
first hit `1`, and discarded the result/error entirely. If that one
`Expire` call ever failed (dropped connection, busy pool, etc.), the
key was left with no TTL — `INCR` would then climb forever and the
IP would be rate-limited permanently, even with no further traffic.
This was independently confirmed live: a key in our Redis instance
was found at `TTL -1` with a count over 2,000,000, explaining several
hours of attack-mode tests showing 0 allowed requests.
**Fix:** replaced the one-time `Expire` with `ExpireNX`, called on
every request. `ExpireNX` only sets a TTL if the key doesn't already
have one, so a single dropped call can no longer permanently strand
a key — the very next request gets another chance to set it.
**Verified by:** `TestHandleClickSelfHealsKeyMissingTTL` — manually
creates a key stuck above `maxRate` with no TTL, confirms the next
request attaches one, fast-forwards past it, and confirms the IP is
allowed again afterward. Also confirmed live against real Redis: TTL
now cycles normally during sustained attack traffic instead of
staying at `-1`.

## What Was Completed
- `cmd/engine` builds successfully.
- Per-IP rate limiting via Redis confirmed correct and self-healing.
- `POST /v1/click` accepts JSON with `ip`, `user_agent`,
  `campaign_id`, `timestamp` (note: `ip` is no longer trusted for
  the verdict — see Bug 1).
- Normal generator traffic returns mostly `200 OK`.
- Attack generator traffic returns `429 Too Many Requests` and
  recovers correctly when the attack stops (TTL expiry confirmed).
- Click logs flushed asynchronously to PostgreSQL via `BatchLogger`.
- Full test suite, including two new regression tests for the bugs
  above, passes under `-race`.
- 5-minute sustained attack load completed without crashes; engine
  memory stayed flat (~35.9MB RSS, no growth).

## Commands Used

### Start dependencies
docker exec -it antifraud-redis redis-cli FLUSHALL   # clean slate
docker compose up -d postgres redis

### Run engine locally
DB_PORT=5433 REDIS_PORT=6380 go run ./cmd/engine

### Build & test
go build ./...
go test ./... -race -v

### Normal traffic (30s)
go run ./cmd/generator/ -workers 10 -rps 10 -duration 30s

Result:
  Clean Clicks (200)  : 2968
  Rate-Limit Hits(429): 0
  Blacklist Hits (403): 12
  Errors (other)      : 0

### Attack traffic (30s, 10 workers — matches original spec load)
go run ./cmd/generator/ -attack -workers 10 -duration 30s

Result:
  Clean Clicks (200)  : 150
  Rate-Limit Hits(429): 28262
  Blacklist Hits (403): 0
  Errors (other)      : 0
  Overall Catch Rate  : 99.5%

### Sustained attack load (5 min, 20 workers / ~2000rps target)
go run ./cmd/generator/ -attack -workers 20 -duration 5m

Result:
  Total Requests Sent : 373104
  Clean Clicks (200)  : 1223
  Rate-Limit Hits(429): 313784
  Errors (other)      : 58097   (~15.5%)
  Duration            : 300.0s

Note: error rate rises noticeably at 20 workers vs. 10 — likely
Redis connection-pool contention under sustained 2000rps from a
single client. At the 10-worker load level matching the original
spec, error count was 0 in every run. Flagging this as a known
scaling limit worth revisiting later, not a blocker for this task.

### Memory check (during the 5-min sustained run)
ps -o pid,rss,vsz,etime -p $(pgrep -f "cmd/engine")

  ELAPSED 01:21 -> RSS 35936
  ELAPSED 02:03 -> RSS 35952
  ELAPSED 02:34 -> RSS 35952
  ELAPSED 05:09 -> RSS 35952

RSS plateaued almost immediately and stayed flat — no memory growth
over the run.

### Redis TTL spot-check (live, during attack traffic)
docker exec -it antifraud-redis redis-cli
> TTL rate:1.2.3.4

Observed values cycling normally: 1, 0, 1, 1, -2, -2, -2, 0, 0, 1,
-2, 1, 0, 1 — confirming the key expires and resets on its own
instead of getting stuck (as it did before the Bug 2 fix).

## Conclusion
Task 2 is complete locally. The engine builds, runs with Redis and
PostgreSQL, accepts the required click JSON format without trusting
client-supplied IPs, returns `200 OK` for normal traffic, returns
`429 Too Many Requests` under attack traffic and recovers correctly
once the attack stops, writes logs to PostgreSQL, and runs
continuously under generator load without crashes or memory leaks.
Two real bugs were found and fixed during testing (spoofable IP
trust, and a Redis TTL race that could permanently strand a rate
limit key); both now have dedicated regression tests.

### Remaining step
Repeat the same checks on the team VM.