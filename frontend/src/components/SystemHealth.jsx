import { useState, useEffect } from 'react';
import axios from 'axios';

// Real health check, replacing the previous hardcoded "all Healthy" mock.
// Two backend health endpoints exist and are already reachable from this
// origin via frontend/nginx/frontend.conf's existing proxies:
//   /engine/health -> engine service:   { redis, postgres, blacklist_loaded }
//   /api/health    -> analytics service: { postgres }
//
// Honest limitation, worth keeping in mind: "Challenge Service" has no
// dedicated probe of its own, the JS challenge store lives in the same
// Redis instance the engine already checks, so its status mirrors the
// engine's redis status rather than a separate real signal we don't have.
function useHealth(intervalMs = 5000) {
  const [health, setHealth] = useState(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const [engineRes, analyticsRes] = await Promise.allSettled([
        axios.get('/engine/health', { timeout: 4000 }),
        axios.get('/api/health', { timeout: 4000 }),
      ]);

      if (cancelled) return;

      const engine = engineRes.status === 'fulfilled' ? engineRes.value.data : null;
      const analytics = analyticsRes.status === 'fulfilled' ? analyticsRes.value.data : null;

      setHealth({
        engineReachable: engineRes.status === 'fulfilled',
        redis: engine?.redis ?? 'unknown',
        enginePostgres: engine?.postgres ?? 'unknown',
        analyticsPostgres: analytics?.postgres ?? 'unknown',
        blacklistLoaded: engine?.blacklist_loaded ?? false,
      });
    }

    load();
    const timer = setInterval(load, intervalMs);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [intervalMs]);

  return health;
}

function SystemHealth() {
  const health = useHealth(5000);

  // healthy: true | false | null (null = still loading / never got a response yet)
  const services = [
    { name: 'API Gateway', healthy: health ? health.engineReachable : null },
    { name: 'Challenge Service', healthy: health ? health.redis === 'healthy' : null },
    { name: 'Redis', healthy: health ? health.redis === 'healthy' : null },
    {
      name: 'PostgreSQL',
      healthy: health ? health.enginePostgres === 'healthy' && health.analyticsPostgres === 'healthy' : null,
    },
    { name: 'Blacklist Sync', healthy: health ? health.blacklistLoaded === true : null },
  ];

  const anyUnknown = services.some((s) => s.healthy === null);
  const allHealthy = !anyUnknown && services.every((s) => s.healthy === true);

  return (
    <div className="border border-border rounded-lg overflow-hidden flex flex-col flex-1">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">System health</h2>
      </div>
      <div className="p-4 flex items-stretch flex-1">
        {/* Left half: services list */}
        <div className="w-1/2 flex flex-col justify-center gap-3 pr-4">
          {services.map((s) => (
            <div key={s.name} className="flex items-center justify-between text-xs">
              <span className="text-text-main">{s.name}</span>
              <span
                className={`flex items-center gap-1.5 font-medium ${s.healthy === null ? 'text-text-muted' : s.healthy ? 'text-success' : 'text-danger'
                  }`}
              >
                <span
                  className={`w-1.5 h-1.5 rounded-full ${s.healthy === null ? 'bg-text-muted' : s.healthy ? 'bg-success' : 'bg-danger'
                    }`}
                />
                {s.healthy === null ? 'Checking...' : s.healthy ? 'Healthy' : 'Down'}
              </span>
            </div>
          ))}
        </div>
        {/* Right half: shield summary, centered */}
        <div className="w-1/2 flex flex-col items-center justify-center gap-1.5 pl-4 border-l border-border">
          <div
            className="w-16 h-16 rounded-xl flex items-center justify-center"
            style={{ backgroundColor: 'var(--color-primary-light)' }}
          >
            <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="var(--color-primary)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              <path d="M9 12l2 2 4-4" />
            </svg>
          </div>
          <span className="text-xs text-text-muted mt-1">All Systems</span>
          <span
            className={`text-base font-bold ${anyUnknown ? 'text-text-muted' : allHealthy ? 'text-primary' : 'text-danger'
              }`}
          >
            {anyUnknown ? 'Checking...' : allHealthy ? 'Operational' : 'Degraded'}
          </span>
        </div>
      </div>
    </div>
  );
}

export default SystemHealth;