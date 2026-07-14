import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

const REASON_SERIES = [
  { key: 'suspicious_agent', color: '#8b7cf6', label: 'User-Agent' },
  { key: 'js_challenge', color: '#38bdf8', label: 'JS Challenge' },
  { key: 'suspicious_headers', color: '#22d3ee', label: 'Header' },
  { key: 'geoip_policy', color: '#f0616d', label: 'Blacklist' },
  { key: 'rate_limit_exceeded', color: '#fbbf24', label: 'Rate Limit' },
];

const JS_REASONS = ['no_js_challenge', 'challenge_too_fast', 'challenge_mismatch'];

function BlockedByReason({ trend }) {
  const chartData = (trend || []).map((d) => {
    const b = d.breakdown || {};
    const jsBlocked = JS_REASONS.reduce((sum, r) => sum + (b[r] || 0), 0);
    return {
      date: d.date?.slice(5) ?? '',
      'User-Agent': b.suspicious_agent || 0,
      'JS Challenge': jsBlocked,
      Header: b.suspicious_headers || 0,
      Blacklist: b.geoip_policy || 0,
      'Rate Limit': b.rate_limit_exceeded || 0,
    };
  });

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Blocked by reason over time</h2>
      </div>

      <div className="p-4">
        <div className="flex items-center justify-center gap-4 mb-3">
          {REASON_SERIES.map((s) => (
            <div key={s.key} className="flex items-center gap-1.5 text-[11px] text-text-muted">
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: s.color }} />
              {s.label}
            </div>
          ))}
        </div>

        <div className="h-[170px]">
          {chartData.length === 0 ? (
            <p className="text-sm text-text-muted text-center pt-14">No blocked data yet.</p>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-chart-bar)" />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: 'var(--color-chart-text)' }} />
                <YAxis tick={{ fontSize: 11, fill: 'var(--color-chart-text)' }} />
                <Tooltip
                  contentStyle={{
                    background: 'var(--color-app-bg)',
                    border: '1px solid var(--color-border)',
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                />
                <Area type="monotone" dataKey="Blacklist" stackId="1" stroke="#f0616d" fill="#f0616d" />
                <Area type="monotone" dataKey="Header" stackId="1" stroke="#22d3ee" fill="#22d3ee" />
                <Area type="monotone" dataKey="JS Challenge" stackId="1" stroke="#38bdf8" fill="#38bdf8" />
                <Area type="monotone" dataKey="Rate Limit" stackId="1" stroke="#fbbf24" fill="#fbbf24" />
                <Area type="monotone" dataKey="User-Agent" stackId="1" stroke="#8b7cf6" fill="#8b7cf6" />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>
    </div>
  );
}

export default BlockedByReason;