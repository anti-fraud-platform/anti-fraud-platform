import { useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchAnalyticsStats } from '../api/analytics'

// Main dashboard page.
// On button click, makes a test request to the analytics endpoint and shows the result.
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
      // The analytics backend is not running yet - this is expected during Week 1.
      setErrorMsg(err?.message || 'Failed to reach the analytics backend')
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
      <p>Connectivity check with the analytics backend (GET /v1/analytics/stats)</p>

      <button onClick={handleCheck} style={{ padding: '0.6rem 1.2rem', cursor: 'pointer' }}>
        Check analytics connection
      </button>

      <div style={{ marginTop: '1.5rem' }}>
        {status === 'loading' && <p>Loading...</p>}

        {status === 'success' && (
          <div>
            <p style={{ color: 'green' }}>Connected! Backend response:</p>
            <pre style={{ background: '#f4f4f4', padding: '1rem', borderRadius: '6px', overflow: 'auto' }}>
              {JSON.stringify(data, null, 2)}
            </pre>
          </div>
        )}

        {status === 'error' && (
          <div>
            <p style={{ color: '#b00' }}>
              Analytics backend is not available yet (this is expected this week).
            </p>
            <p style={{ color: '#888', fontSize: '0.9rem' }}>Details: {errorMsg}</p>
          </div>
        )}
      </div>
    </div>
  )
}

export default Dashboard