import axios from 'axios'

// Базовый клиент. Все пути идут через /api -> прокси Vite перенаправит на бэкенд.
const client = axios.create({
  baseURL: '/api',
  timeout: 5000,
})

// Запрос к ручке аналитики второго бэкенда: GET /v1/analytics/stats
export async function fetchAnalyticsStats() {
  const response = await client.get('/v1/analytics/stats')
  return response.data
}