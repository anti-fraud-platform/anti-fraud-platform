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