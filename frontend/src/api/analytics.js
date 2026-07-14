import axios from 'axios'

// Base client. Requests go through /api -> Vite proxy forwards to the analytics backend (port 8081).
const client = axios.create({
  baseURL: '/api',
  timeout: 5000,
})

// GET /v1/analytics/stats
// Backend returns:
//   total_clicks    - total number of clicks
//   allowed_count   - clicks that were not blocked
//   blocked_count   - clicks flagged as bots
//   blocked_bots    - same as blocked_count (kept for backward compatibility)
//   saved_money_usd - money saved, summed across campaigns using each one's real cost_per_click
//   budget_saved    - same total as saved_money_usd (kept as a separate field for backward compatibility)
//   top_blocked_ips - [{ ip, count }] top offending IPs
//   campaigns       - [{ campaign_id, total_clicks, blocked_bots, saved_money_usd }]
export async function fetchAnalyticsStats() {
  const response = await client.get('/v1/analytics/stats')
  return response.data
}

// GET /v1/analytics/logs
// Query params: page, limit, campaign_id, is_bot, reason
// Backend returns: { data: [...ClickLogEntry], total, page, limit, total_pages }
// ClickLogEntry: id, ip, campaign_id, user_agent, is_bot, reason, processed_at, country, city.
//   country - ISO-3166 alpha-2 code from GeoIP (e.g. "RU"), empty if unresolved.
export async function fetchAnalyticsLogs(params = {}) {
  const response = await client.get('/v1/analytics/logs', { params })
  return response.data
}

// GET /v1/analytics/trend
// Backend returns: { data: [{ date, total_clicks, allowed_count, blocked_count, breakdown }] }
export async function fetchAnalyticsTrend() {
  const response = await client.get('/v1/analytics/trend')
  return response.data
}