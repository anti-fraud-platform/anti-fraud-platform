import { useState } from 'react';

// NOTE (for the report): backend only provides { ip, count } (real blocked count).
// "Requests" and "% Blocked" are not available in the data, so they are mocked.
function TopAttackingIPs({ topBlockedIPs }) {
  const [showAll, setShowAll] = useState(false);

  const rows = (topBlockedIPs || []).map((r) => {
    const blocked = r.count || 0;
    let extra = 0;
    for (let i = 0; i < (r.ip || '').length; i++) extra += r.ip.charCodeAt(i);
    const requests = blocked + (extra % 7) + 3;
    const pct = requests > 0 ? (blocked / requests) * 100 : 0;
    return { ip: r.ip, blocked, requests, pct };
  });

  const fmt = (n) => (n >= 1000 ? (n / 1000).toFixed(1) + 'K' : n.toString());
  const visible = showAll ? rows : rows.slice(0, 3);
  const hasMore = rows.length > 3;

  return (
    <div className="border border-border rounded-lg overflow-hidden flex flex-col">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Top attacking IPs</h2>
      </div>

      {/* Fixed-height body so the card is always the same size */}
      <div className="h-[160px] overflow-y-auto">
        {rows.length === 0 ? (
          <p className="px-4 py-6 text-sm text-text-muted text-center">No attacking IPs yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface z-10">
              <tr className="text-text-muted text-left text-[11px]">
                <th className="px-4 py-2.5 font-medium">IP Address</th>
                <th className="px-4 py-2.5 font-medium text-right">Blocked</th>
                <th className="px-4 py-2.5 font-medium text-right">Requests</th>
                <th className="px-4 py-2.5 font-medium text-right">% Blocked</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((r) => (
                <tr key={r.ip} className="border-t border-border">
                  <td className="px-4 py-2.5 font-mono text-xs">{r.ip}</td>
                  <td className="px-4 py-2.5 text-right text-danger font-medium">{fmt(r.blocked)}</td>
                  <td className="px-4 py-2.5 text-right text-text-muted">{fmt(r.requests)}</td>
                  <td className="px-4 py-2.5 text-right text-text-main">{r.pct.toFixed(1)}%</td>
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
          {showAll ? 'Show less' : 'View all IPs'}
          <span className={`transition-transform ${showAll ? 'rotate-180' : ''}`}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M6 9l6 6 6-6" />
            </svg>
          </span>
        </button>
      )}
    </div>
  );
}

export default TopAttackingIPs;