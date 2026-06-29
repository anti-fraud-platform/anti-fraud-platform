# Task 12 — Extend GET /v1/analytics/stats

## Description
Extended the analytics `GET /v1/analytics/stats` endpoint so the frontend can
render real aggregated counts on initial page load. Added the following fields,
queried directly from Postgres:

- `allowed_count`   — clicks not blocked (total - blocked)
- `blocked_count`   — clicks flagged as bots
- `budget_saved`    — budget saved = blocked_count x fixed CPC ($5)
- `top_blocked_ips` — top 10 offending IPs with block counts

Existing fields (total_clicks, blocked_bots, saved_money_usd, campaigns) kept
for backward compatibility. Fixed CPC extracted into a constant costPerClickUSD = 5.0.

## Build / Run
docker compose up -d
docker ps

Analytics startup log:
Connected to PostgreSQL
Analytics service listening on :8081

## Verification
curl.exe http://localhost:8082/v1/analytics/stats

## Results
Real JSON from live Postgres:
- total_clicks: 2970
- allowed_count: 2960   (2970 - 10)
- blocked_count: 10
- budget_saved: 50      (10 x 5)
- top_blocked_ips: 10 real IPs aggregated from DB
- campaigns: per-campaign breakdown

## Conclusion
GET /v1/analytics/stats returns real counts the frontend can render, computed
directly from Postgres. Verified live with curl against the running stack.