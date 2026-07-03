import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

const SERIES = [
  { key: 'Incoming', color: '#8b7cf6' },
  { key: 'Blocked', color: '#f0616d' },
  { key: 'Allowed', color: '#34d399' },
];

function TrafficOverTime({ trend }) {
  const data = (trend || []).map((d) => ({
    date: d.date?.slice(5) ?? '',
    Incoming: d.total_clicks ?? 0,
    Blocked: d.blocked_count ?? 0,
    Allowed: d.allowed_count ?? 0,
  }));

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      {/* Header: title only */}
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Traffic over time (7 days)</h2>
      </div>

      <div className="p-4">
        {/* Legend above the chart, centered */}
        <div className="flex items-center justify-center gap-4 mb-3">
          {SERIES.map((s) => (
            <div key={s.key} className="flex items-center gap-1.5 text-[11px] text-text-muted">
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: s.color }} />
              {s.key}
            </div>
          ))}
        </div>

        {/* Chart */}
        <div className="h-[170px]">
          {data.length === 0 ? (
            <p className="text-sm text-text-muted text-center pt-14">No trend data yet.</p>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
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
                <Line type="monotone" dataKey="Incoming" stroke="#8b7cf6" strokeWidth={2} dot={{ r: 3 }} />
                <Line type="monotone" dataKey="Blocked" stroke="#f0616d" strokeWidth={2} dot={{ r: 3 }} />
                <Line type="monotone" dataKey="Allowed" stroke="#34d399" strokeWidth={2} dot={{ r: 3 }} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>
    </div>
  );
}

export default TrafficOverTime;