import { useState, useEffect } from 'react'
import { fetchAnalyticsLogs } from '../api/analytics'

// A hook that owns the logs state and polls the backend on an interval.
// params     - params for the endpoint, extracted from props.
// intervalMs - how often to poll (default 2500ms = 2.5s).

// Reused code from useState.

export function useLogs(params = {}, intervalMs = 2500) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    let cancelled = false

    async function load() {
      try {
        const logsResponse = await fetchAnalyticsLogs(params)
        if (cancelled) return
        setData(logsResponse)
        setError(null)
      } catch (err) {
        if (cancelled) return
        // Keep last good data on screen, just flag the error.
        setError(err?.message || 'Failed to reach analytics backend')
        console.error('Analytics polling error:', err)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load() // first call immediately
    const timer = setInterval(load, intervalMs) // then repeat every intervalMs

    // Cleanup: stop polling when component unmounts (prevents leaks).
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [params, intervalMs])

  return { data, loading, error }
}