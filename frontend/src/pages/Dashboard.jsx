import { useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchAnalyticsStats } from '../api/analytics'

// Главная страница дашборда.
// По кнопке делает тестовый запрос к ручке аналитики и показывает результат.
function Dashboard() {
  const [status, setStatus] = useState('idle') // idle | loading | success | error
  const [data, setData] = useState(null)
  const [errorMsg, setErrorMsg] = useState('')

  async function handleCheck() {
    setStatus('loading')
    setErrorMsg('')
    setData(null)
    try {
      const result = await fetchAnalyticsStats()
      setData(result)
      setStatus('success')
    } catch (err) {
      // Бэкенд аналитики ещё не поднят — это ожидаемо на Week 1.
      setErrorMsg(err?.message || 'Не удалось связаться с бэкендом аналитики')
      setStatus('error')
    }
  }

  return (
    <div style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <nav style={{ marginBottom: '1.5rem' }}>
        <Link to="/" style={{ marginRight: '1rem' }}>Dashboard</Link>
        <Link to="/about">About</Link>
      </nav>

      <h1>Anti-Fraud Dashboard</h1>
      <p>Проверка связи с бэкендом аналитики (GET /v1/analytics/stats)</p>

      <button onClick={handleCheck} style={{ padding: '0.6rem 1.2rem', cursor: 'pointer' }}>
        Проверить связь с аналитикой
      </button>

      <div style={{ marginTop: '1.5rem' }}>
        {status === 'loading' && <p>Загрузка…</p>}

        {status === 'success' && (
          <div>
            <p style={{ color: 'green' }}>✅ Связь есть! Ответ бэкенда:</p>
            <pre style={{ background: '#f4f4f4', padding: '1rem', borderRadius: '6px', overflow: 'auto' }}>
              {JSON.stringify(data, null, 2)}
            </pre>
          </div>
        )}

        {status === 'error' && (
          <div>
            <p style={{ color: '#b00' }}>
              ⚠️ Бэкенд аналитики пока недоступен (это нормально на этой неделе).
            </p>
            <p style={{ color: '#888', fontSize: '0.9rem' }}>Детали: {errorMsg}</p>
          </div>
        )}
      </div>
    </div>
  )
}

export default Dashboard