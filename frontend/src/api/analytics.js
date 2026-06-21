import axios from 'axios'

// Base client. Requests go through /api -> Vite proxy forwards to the analytics backend (port 8081).
const client = axios.create({
  baseURL: '/api',
  timeout: 5000,
})

// GET /v1/analytics/stats
// Backend returns: { total_clicks, blocked_bots, saved_money_usd }
export async function fetchAnalyticsStats() {
  const response = await client.get('/v1/analytics/stats')
  return response.data
}

// GET /v1/analytics/logs
// Query params: page, limit, campaign_id, is_bot, reason
// Backend returns: { data: [...ClickLogEntry], total, page, limit, total_pages }
// ClickLogEntry: id, ip, campaign_id, user_agent, is_bot, reason, processed_at.
export async function fetchAnalyticsLogs(params = {}) {
  const response = await client.get('/v1/analytics/logs', { params })
  return response.data
}