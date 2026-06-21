import { useState, useEffect } from 'react'
import { fetchAnalyticsLogs } from '../api/analytics'

export function useLogs(params = {}, intervalMs = 2500) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const { page, limit, campaign_id, is_bot, reason } = params

  useEffect(() => {
    let cancelled = false

    async function load() {
      try {
        const logsResponse = await fetchAnalyticsLogs({ page, limit, campaign_id, is_bot, reason })
        if (cancelled) return
        setData(logsResponse)
        setError(null)
      } catch (err) {
        if (cancelled) return
        setError(err?.message || 'Failed to reach analytics backend')
        console.error('Analytics polling error:', err)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load()
    const timer = setInterval(load, intervalMs)

    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [page, limit, campaign_id, is_bot, reason, intervalMs])

  return { data, loading, error }
}