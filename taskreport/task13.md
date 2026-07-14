# Task 13 — Bug Fixes: Click Simulator + Frontend

## Description

Fixed a critical error on the nginx click simulator page and resolved 9
frontend bugs identified during a full codebase review.

### Critical Error Fixed

**`deployments/nginx/html/index.html`** — `Cannot read properties of undefined (reading 'digest')`

**Root cause:** `crypto.subtle` (Web Crypto API) is only available in secure
contexts (HTTPS or localhost). When the simulator page is served over plain
HTTP on a non-localhost address (e.g. `http://192.168.x.x:9090` via Docker),
`window.crypto.subtle` is `undefined`, causing a hard crash on line 105.

**Fix:** Implemented a dual-path SHA-256 strategy:
- Native `crypto.subtle` used when available (fast, constant-time)
- Pure-JS SHA-256 fallback (RFC 6234) when `crypto.subtle` is unavailable
- Both implementations produce identical hashes (verified against Go server)

### Frontend Bugs Fixed

1. **Dashboard hardcoded deltas** — `Dashboard.jsx` ignored backend fields
   `total_clicks_delta_percent` and `blocked_count_delta_percent`, instead
   rendering hardcoded "18.4%", "24.6%", "11.2%". Now uses real data from
   the analytics API.

2. **Layout hardcoded date** — `Layout.jsx` showed "May 10 - May 16, 2024"
   permanently. Replaced with "Last 7 days".

3. **Skeleton dark mode broken** — `SkeletonCard`, `SkeletonBlacklistRow`,
   `SkeletonChart` used `bg-gray-200` (hardcoded light gray), invisible in
   dark theme. Replaced with `bg-border` (theme-aware).

4. **BlockedByReason mock data** — Chart rendered entirely fake data. Now
   receives real `trend` prop from Dashboard, extracts per-reason breakdown
   from the `/v1/analytics/trend` endpoint.

5. **Blacklist no polling** — `Blacklist.jsx` only retried on error
   (`setTimeout` in `catch`). On success, data never refreshed. Changed to
   `setInterval` so the page polls every 5 seconds.

6. **RecentDetections CSV truncation** — CSV export fetched only 100 records
   (`limit: 100`). Added pagination loop to fetch all pages.

7. **Blacklist CSV no escaping** — `row.join(',')` produced malformed CSV if
   values contained commas. Added `csvEscape()` function matching
   RecentDetections.

8. **useTrend duplicate axios client** — Created its own `axios.create()`
   instead of using the shared client from `api/analytics.js`. Added
   `fetchAnalyticsTrend()` to the shared API module.

9. **Logs page missing Location** — `Logs.jsx` didn't display country/city
   despite the backend returning it. Added Location column with flag emojis
   and Intl.DisplayNames, matching RecentDetections layout.

## Build / Run

```bash
docker compose down -v
docker compose up -d --build
docker ps
```

## Verification

### Simulator (critical fix)

```bash
# Access simulator over plain HTTP from any non-localhost address:
# http://<your-docker-ip>:9090
# Click "Send Real Click (solves JS challenge)"
# Should show: HTTP 200, status: "success"
```

### Frontend dashboard

```bash
# http://localhost:3001
# - Stat cards: deltas should change with real data (not hardcoded)
# - Header: shows "Last 7 days" (not a fixed date)
# - Charts: Blocked by reason shows real per-reason data (not flat mock)
# - Dark mode: skeleton loaders visible (not invisible white blocks)
# - Logs page: Location column with country flags present
# - Blacklist page: data refreshes every 5 seconds
# - CSV exports: all records exported (not truncated to 100)
```

## Results

All 10 issues resolved. Simulator works on HTTP and HTTPS. Frontend uses
real backend data throughout. Dark mode consistent. CSV exports complete
and properly escaped.

## Conclusion

The nginx click simulator now works on any origin (HTTP/HTTPS/localhost)
via pure-JS SHA-256 fallback. Nine frontend bugs fixed: hardcoded data,
dark mode inconsistencies, missing polling, CSV export issues, and
duplicate code.
