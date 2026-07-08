import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

// NOTE: per-reason time series isn't available from the backend yet,
// so this uses representative mock data for the redesign.
const MOCK = [
  { date: 'Day 1', Blacklist: 40, Header: 30, JS: 55, UserAgent: 60 },
  { date: 'Day 2', Blacklist: 45, Header: 28, JS: 60, UserAgent: 58 },
  { date: 'Day 3', Blacklist: 38, Header: 35, JS: 52, UserAgent: 65 },
  { date: 'Day 4', Blacklist: 50, Header: 32, JS: 58, UserAgent: 62 },
  { date: 'Day 5', Blacklist: 42, Header: 30, JS: 63, UserAgent: 68 },
  { date: 'Day 6', Blacklist: 48, Header: 34, JS: 60, UserAgent: 70 },
  { date: 'Day 7', Blacklist: 44, Header: 31, JS: 66, UserAgent: 72 },
];

const SERIES = [
  { key: 'UserAgent', color: '#8b7cf6', label: 'User-Agent' },
  { key: 'JS', color: '#38bdf8', label: 'JS' },
  { key: 'Header', color: '#22d3ee', label: 'Header' },
  { key: 'Blacklist', color: '#f0616d', label: 'Blacklist' },
];

function BlockedByReason() {
  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Blocked by reason over time</h2>
      </div>

      <div className="p-4">
        {/* Legend above chart, centered — same layout as Traffic chart */}
        <div className="flex items-center justify-center gap-4 mb-3">
          {SERIES.map((s) => (
            <div key={s.key} className="flex items-center gap-1.5 text-[11px] text-text-muted">
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: s.color }} />
              {s.label}
            </div>
          ))}
        </div>

        <div className="h-[170px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={MOCK} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
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
              <Area type="monotone" dataKey="JS" stackId="1" stroke="#38bdf8" fill="#38bdf8" />
              <Area type="monotone" dataKey="UserAgent" stackId="1" stroke="#8b7cf6" fill="#8b7cf6" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}

export default BlockedByReason;