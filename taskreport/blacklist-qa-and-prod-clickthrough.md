# Blacklist QA + Production Click-Through (TA point 2)

## Description
Two-part QA task after Vladimir's backend fix (`79c28ad` "backend bug fixes",
`2caab12` "go version fix", both already on `main` as of 2026-07-10):

1. Confirm `Blacklist.jsx` renders the corrected data correctly — the fix
   changed the data source, not just field shapes.
2. Full click-through of every page against the real production URL
   (`http://10.93.26.161:3001`), not localhost.

## What changed in the backend fix
`blacklistIPsHandler` / `blacklistSummaryHandler` in `cmd/analytics/main.go`
now UNION two sources instead of one:
- `dynamic_blacklist` table (IPs auto-promoted after 5+ flagged hits/hour —
  the "backend bug fixes" commit's dynamic auto-block path)
- `click_logs WHERE reason = 'geoip_policy'` (the old static GeoIP/ASN source)

The JSON shape of `BlacklistIPEntry` did **not** change: `ip`, `block_count`,
`first_blocked`, `last_blocked`. No new/renamed/removed fields.

## Part 1 — Blacklist.jsx verification (done locally)

### Method
Stack brought up via `docker compose up -d --build` (local, ports match prod:
3001 frontend, 8082 analytics, 9090 engine simulator). Generated real mixed
data through the actual detection pipeline (not fixtures):
- 6x POST `/click` with a spoofed non-cloud IP (`203.0.113.77`) via
  `X-Forwarded-For` → 6th request returned
  `{"error":"Blocked by dynamic blacklist."}`, confirming the 5-hit
  auto-promotion path.
- 1x POST `/click` with a Cloudflare-range IP (`104.16.132.229`) →
  `{"error":"Blocked by GeoIP / ASN policy."}`.

Resulting `/api/v1/analytics/blacklist/ips`:
```json
{"items":[
  {"ip":"104.16.132.229","block_count":1,"first_blocked":"2026-07-10 12:34","last_blocked":"2026-07-10 12:34"},
  {"ip":"203.0.113.77","block_count":1,"first_blocked":"2026-07-10 12:34","last_blocked":"2026-07-10 12:34"}
],"total":2}
```

### Result: renders without crashing, but two real issues found

1. **Mislabeled as GeoIP-only.** `Blacklist.jsx` hardcodes GeoIP-specific
   copy everywhere: page title `"GeoIP / ASN Blocks"`, subtitle `"Showing N
   IPs blocked by GeoIP policy"`, empty state `"No GeoIP policy blocks
   found"`, console error `"Error fetching GeoIP policy blocks"`, CSV
   filename `geoip_policy_export_*.csv`. After the fix, the list also
   contains dynamic-auto-blacklist IPs that were never touched by GeoIP
   policy — `203.0.113.77` in the test above is a clean-ASN IP blocked
   purely by hit-count, yet the page tells the user it's a GeoIP block.
2. **Backend can't tell the two apart even if the frontend wanted to.**
   `BlacklistIPEntry` has no `source`/`reason` field — the SQL `UNION ALL ...
   GROUP BY ip` collapses both origins before the Go handler ever sees which
   table a row came from. Fixing the frontend copy alone isn't enough; the
   API needs a discriminator column if we want per-row accuracy (e.g. a
   `source: "geoip_policy" | "dynamic_blacklist"` field, or splitting into
   two response arrays).

### Fix applied in this branch
- `cmd/analytics/main.go`: `BlacklistIPEntry` gained a `source` field.
  `blacklistIPsHandler`'s SQL now tags each sub-select (`'dynamic_blacklist'`
  / `'geoip_policy'`) and uses `STRING_AGG(DISTINCT source, ',')` so an IP
  blocked by both shows both.
- `frontend/src/pages/Blacklist.jsx`: page title changed from
  `"GeoIP / ASN Blocks"` to source-neutral `"Blocked IPs"`, added a
  "Source" column (humanized via `formatSource()`), updated empty state,
  fetch-error copy, and CSV export (filename + new Source column).
- `frontend/src/components/SkeletonBlacklistRow.jsx`: added a skeleton cell
  for the new column.

Re-verified locally after the fix with fresh mixed traffic (6x flagged
clicks from `198.51.100.42` + 1x Cloudflare-range click):
```json
{"items":[
  {"ip":"104.16.132.229","block_count":2,...,"source":"geoip_policy"},
  {"ip":"198.51.100.42","block_count":1,...,"source":"dynamic_blacklist"},
  {"ip":"203.0.113.77","block_count":1,...,"source":"dynamic_blacklist"}
],"total":4}
```
Each row now correctly reports its own source.

### Environment note (unrelated to the bug, worth recording)
Local Windows checkout with `core.autocrlf=true` turns
`deployments/nginx/docker-entrypoint-with-resolver.sh` into CRLF on disk,
which breaks the frontend container (`exec ...: no such file or directory`
— the `#!/bin/sh\r` shebang doesn't resolve on Alpine). The committed blob
is LF-only, so this is a local-checkout artifact, not a repo bug — but it
means `docker compose up` fails out of the box on a fresh Windows clone
without `dos2unix`/WSL checkout settings. Worth a `.gitattributes` entry
(`*.sh text eol=lf`) if this bites others; not added here since it's outside
the scope of the two QA tasks. Not committed as part of this branch.

## Part 2 — Production click-through checklist (http://10.93.26.161:3001)

Run through each page live against prod, not localhost. Local stack behavior
observed above is the baseline to diff against — note anything that looks
different.

### Dashboard (`/`)
- [ ] 4 stat cards populate (Total clicks, Blocked bots, Money saved, Budget
      saved) — not stuck on skeleton/0
- [ ] "Top blocked IPs" table populates
- [ ] Campaign performance table populates
- [ ] Charts (trend, detection pipeline, blocked-by-reason) render with data
- [ ] System Health widget: GeoIP Databases shows healthy (this was the
      `geoResolver` shadowing bug from `8f3cd48`/`328c9ef` — confirm the fix
      actually shows healthy in prod, not just locally)
- [ ] Polling (every 2.5s) doesn't spam errors in the browser console
- [ ] Theme toggle works

### Logs (`/logs`)
- [ ] Table loads and paginates
- [ ] Reason filter dropdown works, including `geoip_policy` option
- [ ] Filtering actually narrows results (server-side filtering — this was
      touched in `49f9620 fix(frontend): server-side filtering in
      RecentActivity`, confirm no regression)

### Blacklist (`/blacklist`)
- [ ] Loads without the "Failed to load GeoIP policy data" error state
- [ ] Table shows real prod IPs, not empty (unless prod genuinely has no
      blocks — check summary counts first)
- [ ] Compare row count / values against what makes sense given prod traffic
      volume
- [ ] CSV export downloads and opens correctly
- [ ] Judge whether the GeoIP-only labeling (see Part 1 finding) is
      confusing on real prod data

### Cross-cutting / prod-vs-local diffs to watch for
- [ ] Mixed content / CORS errors in console (prod is plain `http://`, not
      `https://` — check for browser warnings)
- [ ] Response times noticeably different from local (real network, real DB
      size)
- [ ] Any stale/cached frontend build (hard-refresh to rule out CDN/browser
      cache serving an old bundle)
- [ ] Timezone/date formatting on `first_blocked`/`last_blocked` — server
      formats in Go with no explicit TZ (`time.Format("2006-01-02 15:04")`),
      confirm it reads sensibly against prod server time

## Status
Part 1: done, mislabeling bug found and fixed (source field + neutral
copy), re-verified locally with real mixed traffic.
Part 2: production click-through run by lead against
`http://10.93.26.161:3001` — no issues found; general navigation,
data population and console all checked out clean.
