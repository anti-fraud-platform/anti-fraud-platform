import { PieChart, Pie, Cell, ResponsiveContainer } from 'recharts';

// Human-readable labels for each reason key.
function labelFor(reason) {
  const map = {
    allowed: 'Allowed',
    suspicious_agent: 'Suspicious UA',
    no_js_challenge: 'No JS challenge',
    challenge_too_fast: 'Challenge too fast',
    challenge_mismatch: 'Challenge mismatch',
    suspicious_headers: 'Suspicious headers',
    static_blacklist: 'Static blacklist',
    rate_limit_exceeded: 'Rate limit exceeded',
  };
  return map[reason] || reason;
}

// Fixed palette so colors are stable across renders.
const COLORS = [
  '#8b7cf6', '#38bdf8', '#34d399', '#f0616d',
  '#fbbf24', '#a78bfa', '#22d3ee', '#fb7185',
];

function ReasonBreakdownChart({ reasonBreakdown }) {
  const entries = Object.entries(reasonBreakdown || {})
    .filter(([, v]) => v > 0)
    .sort((a, b) => b[1] - a[1]);

  const total = entries.reduce((sum, [, v]) => sum + v, 0);

  const chartData = entries.map(([reason, value], i) => ({
    name: labelFor(reason),
    value,
    color: COLORS[i % COLORS.length],
  }));

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Reason breakdown</h2>
      </div>

      {total === 0 ? (
        <p className="px-4 py-6 text-sm text-text-muted text-center">
          No blocked clicks yet.
        </p>
      ) : (
        <div className="flex items-center gap-4 p-4">
          {/* Donut */}
          <div className="relative w-[160px] h-[160px] flex-shrink-0">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={chartData}
                  dataKey="value"
                  nameKey="name"
                  innerRadius={52}
                  outerRadius={78}
                  paddingAngle={2}
                  stroke="none"
                >
                  {chartData.map((entry) => (
                    <Cell key={entry.name} fill={entry.color} />
                  ))}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
            {/* Center total */}
            <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
              <span className="text-lg font-semibold text-text-main">
                {total.toLocaleString('en-US')}
              </span>
              <span className="text-xs text-text-muted">Blocked</span>
            </div>
          </div>

          {/* Legend */}
          <div className="flex-1 flex flex-col gap-1.5">
            {chartData.map((entry) => {
              const pct = ((entry.value / total) * 100).toFixed(1);
              return (
                <div key={entry.name} className="flex items-center text-xs">
                  <span
                    className="w-2.5 h-2.5 rounded-sm mr-2 flex-shrink-0"
                    style={{ backgroundColor: entry.color }}
                  />
                  <span className="text-text-muted flex-1 truncate">{entry.name}</span>
                  <span className="text-text-main font-medium ml-2">
                    {entry.value.toLocaleString('en-US')}
                  </span>
                  <span className="text-text-muted ml-2 w-10 text-right">{pct}%</span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

export default ReasonBreakdownChart;