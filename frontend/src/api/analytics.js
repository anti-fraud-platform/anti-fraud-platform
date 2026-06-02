import axios from 'axios'

// Base client. All paths go through /api -> Vite proxy forwards them to the backend.
const client = axios.create({
  baseURL: '/api',
  timeout: 5000,
})

// Request to the analytics backend endpoint: GET /v1/analytics/stats
export async function fetchAnalyticsStats() {
  const response = await client.get('/v1/analytics/stats')
  return response.data
}