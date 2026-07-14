import { useState } from 'react';

function formatUSD(n) {
  return `$${Number(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

// avgCpc is derived (saved_money_usd / blocked_bots) rather than read from a
// dedicated field, but the backend computes saved_money_usd from the real
// per-campaign cost_per_click, so this is exact, not an estimate.
function CampaignCostBreakdown({ campaigns }) {
  const [showAll, setShowAll] = useState(false);

  const rows = (campaigns || [])
    .map((c) => {
      const blocked = c.blocked_bots || 0;
      const saved = c.saved_money_usd || 0;
      const avgCpc = blocked > 0 ? saved / blocked : 0;
      return { id: c.campaign_id, blocked, saved, avgCpc };
    })
    .sort((a, b) => b.saved - a.saved);

  const visible = showAll ? rows : rows.slice(0, 3);
  const hasMore = rows.length > 3;
  const totalSaved = rows.reduce((sum, r) => sum + r.saved, 0);

  return (
    <div className="border border-border rounded-lg overflow-hidden flex flex-col">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Cost saved by campaign</h2>
      </div>

      {/* Fixed-height body so the card doesn't grow with "View all campaigns" */}
      <div className="h-[170px] overflow-y-auto">
        {rows.length === 0 ? (
          <p className="px-4 py-6 text-sm text-text-muted text-center">No campaign cost data yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface z-10">
              <tr className="text-text-muted text-left text-[11px]">
                <th className="px-4 py-2.5 font-medium">Campaign</th>
                <th className="px-4 py-2.5 font-medium text-right">Blocked</th>
                <th className="px-4 py-2.5 font-medium text-right">Cost per click</th>
                <th className="px-4 py-2.5 font-medium text-right">Saved</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((r) => (
                <tr key={r.id} className="border-t border-border">
                  <td className="px-4 py-2.5 font-mono text-xs truncate max-w-[80px]">{r.id}</td>
                  <td className="px-4 py-2.5 text-right text-text-main font-medium">
                    {r.blocked.toLocaleString('en-US')}
                  </td>
                  <td className="px-4 py-2.5 text-right text-text-muted">{formatUSD(r.avgCpc)}</td>
                  <td className="px-4 py-2.5 text-right text-success font-semibold">{formatUSD(r.saved)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {hasMore && (
        <button
          type="button"
          onClick={() => setShowAll((v) => !v)}
          className="w-full py-2.5 text-xs font-medium text-primary border-t border-border hover:bg-primary-light transition-colors flex items-center justify-center gap-1"
        >
          {showAll ? 'Show less' : 'View all campaigns'}
          <span className={`transition-transform ${showAll ? 'rotate-180' : ''}`}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M6 9l6 6 6-6" />
            </svg>
          </span>
        </button>
      )}

      {rows.length > 0 && (
        <div className="flex items-center justify-between px-4 py-2.5 border-t border-border bg-surface grow">
          <span className="text-xs font-medium text-text-muted">Total saved</span>
          <span className="text-sm font-bold text-success">{formatUSD(totalSaved)}</span>
        </div>
      )}
    </div>
  );
}

export default CampaignCostBreakdown;
