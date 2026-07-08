import { useState, useEffect } from 'react';
import axios from 'axios';

const client = axios.create({ baseURL: '/api', timeout: 5000 });

// Polls /v1/analytics/trend for the 7-day time series.
export function useTrend(intervalMs = 5000) {
  const [trend, setTrend] = useState([]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const res = await client.get('/v1/analytics/trend');
        if (!cancelled) setTrend(res.data?.data ?? []);
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