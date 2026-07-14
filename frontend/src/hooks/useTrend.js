import { useState, useEffect } from 'react';
import { fetchAnalyticsTrend } from '../api/analytics';

// Polls /v1/analytics/trend for the 7-day time series.
export function useTrend(intervalMs = 5000) {
  const [trend, setTrend] = useState([]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const res = await fetchAnalyticsTrend();
        if (!cancelled) setTrend(res.data ?? []);
      } catch {
        // keep previous data on error
      }
    }
    load();
    const timer = setInterval(load, intervalMs);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [intervalMs]);

  return trend;
}
