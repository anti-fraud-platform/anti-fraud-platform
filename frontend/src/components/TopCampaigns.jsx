import { useState } from 'react';

function TopCampaigns({ campaigns }) {
  const [showAll, setShowAll] = useState(false);

  const rows = (campaigns || [])
    .map((c) => {
      const total = c.total_clicks || 0;
      const blocked = c.blocked_bots || 0;
      const pct = total > 0 ? (blocked / total) * 100 : 0;
      return { id: c.campaign_id, blocked, pct };
    })
    .sort((a, b) => b.blocked - a.blocked);

  const maxBlocked = rows.reduce((m, r) => Math.max(m, r.blocked), 0) || 1;
  const visible = showAll ? rows : rows.slice(0, 5);
  const hasMore = rows.length > 5;

  return (
    <div className="border border-border rounded-lg overflow-hidden flex flex-col">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Top campaigns by blocked clicks</h2>
      </div>

      {rows.length === 0 ? (
        <p className="px-4 py-6 text-sm text-text-muted text-center">No campaign data yet.</p>
      ) : (
        <>
          <div className="p-4 flex flex-col gap-2.5 flex-1">
            <div className="flex items-center text-[10px] text-text-muted uppercase tracking-wide">
              <span className="flex-1">Campaign</span>
              <span className="w-14 text-right">Blocked</span>
              <span className="w-16 text-right">% Blocked</span>
            </div>
            {visible.map((r) => (
              <div key={r.id} className="flex items-center gap-2 text-xs">
                <span className="font-mono text-text-main w-24 truncate flex-shrink-0">{r.id}</span>
                <div className="w-1/2 h-1.5 rounded-full bg-chart-bar overflow-hidden">
                  <div
                    className="h-full rounded-full bg-primary"
                    style={{ width: `${(r.blocked / maxBlocked) * 100}%` }}
                  />
                </div>
                <span className="w-14 text-right text-text-main font-medium">
                  {r.blocked.toLocaleString('en-US')}
                </span>
                <span className="w-16 text-right text-text-muted">{r.pct.toFixed(1)}%</span>
              </div>
            ))}
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
        </>
      )}
    </div>
  );
}

export default TopCampaigns;