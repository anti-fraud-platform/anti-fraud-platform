// Horizontal bars showing what % of total clicks each detection layer blocked.
// Uses reason_breakdown (already in the stats response) + total clicks.

const LAYERS = [
  { key: 'suspicious_agent', label: 'User-Agent Check', color: '#8b7cf6' },
  { key: 'js_challenge', label: 'JS Challenge', color: '#38bdf8' },
  { key: 'suspicious_headers', label: 'Header Analysis', color: '#22d3ee' },
  { key: 'static_blacklist', label: 'Static Blacklist', color: '#f0616d' },
  { key: 'rate_limit_exceeded', label: 'Rate Limiter', color: '#fbbf24' },
];

// JS Challenge groups three reason values together.
const JS_REASONS = ['no_js_challenge', 'challenge_too_fast', 'challenge_mismatch'];

function PipelineEffectiveness({ reasonBreakdown, totalClicks }) {
  const rb = reasonBreakdown || {};
  const total = totalClicks || 0;

  const rows = LAYERS.map((layer) => {
    let count;
    if (layer.key === 'js_challenge') {
      count = JS_REASONS.reduce((sum, r) => sum + (rb[r] || 0), 0);
    } else {
      count = rb[layer.key] || 0;
    }
    const pct = total > 0 ? (count / total) * 100 : 0;
    return { ...layer, count, pct };
  });

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Pipeline effectiveness</h2>
      </div>
      <div className="p-4 flex flex-col gap-3">
        {rows.map((row, idx) => (
          <div key={row.key} className="flex items-center gap-3 text-xs">
            <span className="w-4 h-4 rounded-full bg-surface text-text-muted flex items-center justify-center text-[10px] flex-shrink-0">
              {idx + 1}
            </span>
            <span className="text-text-main w-28 flex-shrink-0 truncate">{row.label}</span>
            <div className="flex-1 h-2 rounded-full bg-chart-bar overflow-hidden">
              <div
                className="h-full rounded-full"
                style={{ width: `${Math.min(row.pct, 100)}%`, backgroundColor: row.color }}
              />
            </div>
            <span className="text-text-muted w-12 text-right flex-shrink-0">
              {row.pct.toFixed(1)}%
            </span>
          </div>
        ))}
        <p className="text-[11px] text-text-muted mt-1">
          % of total traffic blocked at this layer
        </p>
      </div>
    </div>
  );
}

export default PipelineEffectiveness;