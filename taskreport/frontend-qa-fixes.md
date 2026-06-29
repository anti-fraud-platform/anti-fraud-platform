# Frontend QA Fixes (from lead manual QA)

## Description
Fixed frontend issues found during the lead's manual QA pass.

### Fixed
1. Dashboard now renders budget_saved and top_blocked_ips.
   These fields already existed in GET /v1/analytics/stats (added in Task 12),
   but Dashboard.jsx only read the three legacy fields. Added:
   - a fourth stat card "Budget saved" (reads budget_saved)
   - a "Top blocked IPs" table (reads top_blocked_ips: ip + block count),
     shown next to the existing "Campaign performance" table.

2. Updated stale API comment in analytics.js.
   The comment listed only the 3 old fields. Now documents all current fields:
   total_clicks, allowed_count, blocked_count, blocked_bots, saved_money_usd,
   budget_saved, top_blocked_ips, campaigns.

### Not addressed here (needs backend)
3. Blacklist page is still mock data. Blacklist.jsx reads a hardcoded array
   from ../data/mockData. Showing live blocked IPs needs a backend endpoint
   (e.g. /v1/blacklist, or reading click_logs WHERE reason='static_blacklist')
   which does not exist yet. Pending backend work.

## Build / Run
docker compose up -d --build
(frontend published locally on port 3005 for verification; compose port edit kept local, not committed)

## Verification
Opened the dashboard at http://localhost:3005 with the stack running and the
database populated by the traffic generator.

Observed:
- 4 stat cards including the new "Budget saved" ($60)
- new "Top blocked IPs" table with real IPs from the database
- existing "Campaign performance" table unaffected
- Total clicks 4470 (real data from Postgres)

## Result
Issues #2 and #3 fixed and verified live. Issue #1 (Blacklist) pending backend endpoint.