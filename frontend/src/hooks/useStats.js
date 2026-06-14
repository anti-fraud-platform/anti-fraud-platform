import { useState, useEffect } from 'react'
import { fetchAnalyticsStats } from '../api/analytics'

// Custom hook that owns the stats state and polls the backend on an interval.
// State lives here ("lifted up"); Dashboard reads it and passes values down as props.
// intervalMs - how often to poll (default 2500ms = 2.5s).
export function useStats(intervalMs = 2500) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true) // true only until the first successful load
  const [error, setError] = useState(null)

  useEffect(() => {
    let cancelled = false

    async function load() {
      try {
        const stats = await fetchAnalyticsStats()
        if (cancelled) return
        setData(stats)
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
  }, [intervalMs])

  return { data, loading, error }
}