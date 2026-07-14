# Campaign Cost Dashboard (layout/design phase, pre-Vladimir API)

## Description
Task: wire real per-campaign cost data into the dashboard, replacing wherever
the old flat-$5 assumption was showing. Vladimir's cost-per-click API is the
hard dependency (expected Day 1 EOD) and wasn't ready yet, so this pass is
the layout/design work done ahead of it, plus a Grafana link if time allowed
(not reached — see Status).

## What changed

### New: `CampaignCostBreakdown.jsx`
No card showing cost-per-campaign existed on the dashboard before this —
`saved_money_usd` / `budget_saved` were already returned by
`/v1/analytics/stats` but never rendered anywhere in the frontend. Added a
new card (`frontend/src/components/CampaignCostBreakdown.jsx`) showing, per
campaign: blocked clicks, an estimated cost-per-click (derived client-side as
`saved_money_usd / blocked_bots` until Vladimir's endpoint provides a real
per-campaign CPC), and money saved. A "Total saved" footer row sums the real
per-campaign values. Placed in the left column's chart row
(`Dashboard.jsx`), alongside Traffic and Blocked-by-reason.

TODO left in the component: once the real CPC endpoint lands, replace the
derived `avgCpc` with the backend's real per-campaign value and drop the
"estimated" framing.

### Real per-campaign total vs. the flat-$5 bug
While building the card's total, found the exact bug this task is about:
`budget_saved` in `/v1/analytics/stats` is computed as
`blockedCount(all clicks) * 5.0` (`cmd/analytics/main.go`, flat-rate,
global), completely independent of the per-campaign `saved_money_usd` sum
(which already respects `campaigns.cost_per_click` via `COALESCE(...,
5.00)`). On the current dataset that's $405 (flat) vs $315 (real per-campaign
sum) — an 18-click gap caused by a **separate bug**: the per-campaign SQL
scan (`main.go`, `campID` scanned into a plain `string`) silently drops rows
where `campaign_id IS NULL`, so blocked clicks with no campaign never make it
into the `campaigns[]` array even though they're counted in the global
`blockedCount`. The new card's total now sums the real per-campaign rows
instead of trusting `budget_saved`. **Not fixed**: the NULL-campaign_id
silent-drop in the Go scan loop — flagged here, left for the backend pass
once Vladimir's endpoint lands, since it's a `main.go` change out of scope
for a layout-only pass.

### Real bugs found and fixed along the way (not mocks, not layout)

1. **`TopAttackingIPs.jsx` was showing fabricated numbers, not stale ones.**
   It read `r.count`, a field the API has never returned (the real field is
   `blocked`) — so the "Blocked" column was always 0, and "Requests" / "%
   Blocked" were backfilled with a deterministic hash of the IP string
   (`blocked + (charCodeSum % 7) + 3`). The API already returns real
   `total_requests` per IP (`cmd/analytics/main.go` — `BlockedIPStat`).
   Fixed to read `r.blocked` / `r.total_requests` directly; removed the mock
   entirely.

2. **`RecentDetections.jsx` "Location" column was 100% mocked.** A
   `countryFor(ip)` helper hashed the IP string into one of 7 hardcoded
   countries — completely disconnected from where the click actually came
   from (e.g. a real Russian IP could show "United Kingdom"). The backend
   already runs every click through MaxMind GeoLite2
   (`internal/geoiputil/resolver.go`) and stores the result in
   `click_logs.country` / `click_logs.city`, but `/v1/analytics/logs` never
   selected those columns. Fixed end-to-end:
   - `cmd/analytics/main.go`: `ClickLogEntry` gained `Country`/`City`
     (`sql.NullString` scan, empty string when GeoIP couldn't resolve — e.g.
     private IPs, RFC 5737 test ranges, or anycast blocks MaxMind doesn't
     attribute to one country).
   - `frontend/src/components/RecentDetections.jsx`: renders the real ISO
     country via `Intl.DisplayNames`, flag emoji generated from the ISO code
     (regional-indicator codepoints, not a hardcoded list), city if present,
     "—" if GeoIP genuinely has nothing (verified against the real mmdb —
     see below, this is correct behavior, not a bug).
   - CSV export updated to match.

3. **`TopCampaigns.jsx` column misalignment.** The header row used `flex-1`
   on "Campaign" with no gap, while data rows used a fixed `w-24` name plus a
   `w-1/2` progress bar with `gap-2` — different box models, so "Blocked" /
   "% Blocked" never lined up under their headers. Rebuilt both rows with
   identical fixed-width columns and gaps, bar changed to `flex-1` so it
   always fills the remaining space instead of a fixed 50%.

### Layout / sizing iteration (visual, driven by live docker-compose checks)
- `CampaignCostBreakdown` moved between the right rail and the left chart
  row twice while settling on final placement (left row, swapped with
  `TopCampaigns`); ended with `TopCampaigns` also moved to the left column
  (full-width, below the chart row) since its narrow-rail sizing no longer
  fit its content once it needed 4-column layout math.
- Both `TopCampaigns` and `CampaignCostBreakdown` bodies are now fixed-height
  + `overflow-y-auto` (matching the existing `TopAttackingIPs` pattern) so
  neither card grows unbounded as campaigns are added — `TopCampaigns` sized
  for exactly 4 visible rows (`h-[160px]`, `slice(0, 4)`), consistent with
  its "View all campaigns" toggle.
- `RecentDetections` page size reduced 8 → 5 rows so its card height roughly
  matches `SystemHealth` in the right rail.

## Verification method
All changes verified against a **local docker-compose stack**
(`docker compose up --build`, 6 containers: postgres, redis, engine,
analytics, nginx-engine, frontend), not `npm run dev` — rebuilt after every
change and re-checked via:
- `curl` against the real `/v1/analytics/stats` and `/v1/analytics/logs`
  endpoints to confirm actual payload shape/values before touching frontend
  code that reads them.
- Grepping the built frontend JS bundle for expected strings after each
  rebuild, to confirm the change actually shipped (not just compiled).
- A standalone Go program (`internal/geoiputil.OpenBestEffort` against the
  real `geoip/*.mmdb` files) to independently confirm the GeoIP "—" cases are
  correct MaxMind behavior, not a scanning bug — see finding #2 above.
- Visual review via a browser session pointed at the running
  `localhost:3001` container after each rebuild.

### Environment note
Local Windows checkout (`core.autocrlf=true`) turns several `.sh` files
(`deployments/nginx/docker-entrypoint-with-resolver.sh` and 11 files under
`scripts/`) into CRLF on disk, which breaks the frontend container build
(`exec ...: no such file or directory`). This was already flagged in
`taskreport/blacklist-qa-and-prod-clickthrough.md` as "worth a
`.gitattributes` entry, not added since out of scope." Same call made here —
line endings were normalized locally (uncommitted) to unblock testing on
this branch, but the fix isn't included in this commit; still open for
whoever picks it up next.

## Not done in this pass
- **Real per-campaign CPC from Vladimir's endpoint.** Blocked on the
  dependency landing; `avgCpc` in `CampaignCostBreakdown.jsx` is a derived
  estimate (`saved_money_usd / blocked_bots`) until then.
- **Grafana widget/link.** Named as "if there's time" in the task; not
  reached this pass.
- **NULL `campaign_id` silent-drop bug** (`main.go`, per-campaign scan) —
  documented above, not fixed.
- **Production click-through** (`http://10.93.26.161:3001`). Couldn't reach
  the university VM from this environment (connection timeout). Everything
  above was verified against a local docker-compose stack with real
  pipeline traffic, not localhost `npm run dev` — but per the same standard
  applied to the blacklist fix, this still needs a real prod click-through
  before calling the layout phase fully done. Flagging as an explicit open
  item rather than claiming it.

## Status
Layout/design phase done and verified locally against a full docker-compose
stack with real traffic through the actual pipeline (not fixtures, not
`npm run dev`). Three real (non-mock, non-layout) bugs found and fixed along
the way — see "Real bugs found and fixed" above. Waiting on Vladimir's
cost-per-click API to wire in real per-campaign CPC. Production
click-through still outstanding (VM unreachable from this environment).
