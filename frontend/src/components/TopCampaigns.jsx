function TopCampaigns({ campaigns }) {
  const rows = (campaigns || [])
    .map((c) => {
      const total = c.total_clicks || 0;
      const blocked = c.blocked_bots || 0;
      const pct = total > 0 ? (blocked / total) * 100 : 0;
      return { id: c.campaign_id, blocked, pct };
    })
    .sort((a, b) => b.blocked - a.blocked);

  const maxBlocked = rows.reduce((m, r) => Math.max(m, r.blocked), 0) || 1;

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Top campaigns by blocked clicks</h2>
      </div>
      {rows.length === 0 ? (
        <p className="px-4 py-6 text-sm text-text-muted text-center">No campaign data yet.</p>
      ) : (
        <div className="p-4 flex flex-col gap-3">
          {/* header row */}
          <div className="flex items-center text-[11px] text-text-muted uppercase tracking-wide">
            <span className="flex-1">Campaign</span>
            <span className="w-16 text-right">Blocked</span>
            <span className="w-16 text-right">% Blocked</span>
          </div>
          {rows.map((r) => (
            <div key={r.id} className="flex items-center gap-3 text-xs">
              <span className="font-mono text-text-main w-32 truncate flex-shrink-0">{r.id}</span>
              <div className="flex-1 h-2 rounded-full bg-chart-bar overflow-hidden">
                <div
                  className="h-full rounded-full bg-primary"
                  style={{ width: `${(r.blocked / maxBlocked) * 100}%` }}
                />
              </div>
              <span className="w-16 text-right text-text-main font-medium">
                {r.blocked.toLocaleString('en-US')}
              </span>
              <span className="w-16 text-right text-text-muted">{r.pct.toFixed(1)}%</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default TopCampaigns;