# Frontend Redesign — Dashboard (Night Ops)

## What this is
The dashboard was redesigned to match the "Night Ops" mockup:
a dark theme with a toggle, metric cards, a detection pipeline,
charts, tables, and side panels.

All layout and styling is done. Most of the data is REAL (from the
analytics API). A few parts show FAKE (mock) numbers because the
backend doesn't provide that data yet. This report lists exactly
what is fake and what we need to make it real.

## Real data (already working)
These read live data from `/v1/analytics/stats`, `/trend`, and `/logs`:
- Total clicks, Blocked clicks, Allowed clicks, Money saved
- Detection pipeline stages (User-Agent, JS Challenge, Header, Blacklist, Rate Limiter, Allowed)
- Reason breakdown donut
- Pipeline effectiveness bars
- Top campaigns by blocked clicks
- Recent detections table (with pagination)
- Recent activity feed
- Traffic over time (line chart) — real, but shows only 1 day right now because there is only 1 day of data

## Fake (mock) data — needs backend to become real
Each item below is fake ONLY because the backend has no such data yet.

1. **Stat card deltas ("▲ 18.4% vs prev 7 days")**
   - Fake: the little percentages under each number.
   - Needs: the API should return how each number changed vs the previous 7 days.

2. **"Blocked by reason over time" chart (the colorful area chart)**
   - Fake: the whole chart uses made-up numbers.
   - Needs: an API that gives blocked counts per reason PER DAY (a time series).

3. **"Location" column in Recent detections (country + flag)**
   - Fake: the country is guessed from the IP, not real.
   - Needs: geo-lookup by IP on the backend (which country each IP is from).

4. **"Method" column in Recent detections (always shows GET)**
   - Fake: always "GET".
   - Needs: the logs should store the real HTTP method of each request.

5. **Top attacking IPs — "Requests" and "% Blocked" columns**
   - Fake: only "Blocked" is real. Requests and % are made up.
   - Needs: the API should return total requests per IP (not just blocked count).

6. **System health (API Gateway, Redis, PostgreSQL... "Healthy")**
   - Fake: all statuses are hard-coded as "Healthy".
   - Needs: a health-check endpoint that actually pings each service.

## Notes
- Everything works in both light and dark theme (Logs and Blacklist pages fixed too).
- Blacklist page uses REAL data from `/v1/analytics/blacklist/ips`.
- Once the backend adds the data above, we just swap the mock values for real API calls — the design stays the same.