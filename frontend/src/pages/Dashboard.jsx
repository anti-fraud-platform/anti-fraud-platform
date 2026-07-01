import Layout from '../components/Layout';
import { useStats } from '../hooks/useStats';
import StatCard from '../components/StatCard';
import SkeletonCard from '../components/SkeletonCard';
import SkeletonChart from '../components/SkeletonChart';

function formatNumber(n) {
  return Number(n).toLocaleString('en-US');
}
function formatMoney(n) {
  return '$' + Number(n).toLocaleString('en-US', { maximumFractionDigits: 0 });
}

function formatReasonLabel(reason) {
  switch (reason) {
    case 'allowed':
      return 'Allowed';
    case 'suspicious_agent':
      return 'Suspicious UA';
    case 'no_js_challenge':
      return 'No JS challenge';
    case 'challenge_too_fast':
      return 'Challenge solved too fast';
    case 'challenge_mismatch':
      return 'Challenge mismatch';
    case 'suspicious_headers':
      return 'Suspicious headers';
    case 'static_blacklist':
      return 'Static blacklist';
    case 'rate_limit_exceeded':
      return 'Rate limit exceeded';
    default:
      return reason;
  }
}

function Dashboard() {
  const { data, loading, error } = useStats(2500);

  const statItems = data
    ? [
      { label: 'Total clicks', value: formatNumber(data.total_clicks), danger: false },
      { label: 'Blocked bots', value: formatNumber(data.blocked_bots), danger: true },
      { label: 'Budget saved', value: formatMoney(data.budget_saved), danger: false },
      { label: 'JS challenge blocks', value: formatNumber(data.js_challenge_blocked ?? 0), danger: true },
      { label: 'Header heuristic blocks', value: formatNumber(data.header_heuristic_blocked ?? 0), danger: true },
    ]
    : [];

  const campaigns = data?.campaigns ?? [];
  const topBlockedIPs = data?.top_blocked_ips ?? [];
  const reasonBreakdown = data?.reason_breakdown ?? {};
  const reasonEntries = Object.entries(reasonBreakdown).sort((a, b) => b[1] - a[1]);

  return (
    <Layout title="Dashboard">
      {loading && !data && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-4">
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <SkeletonChart />
            <SkeletonChart />
          </div>
        </>
      )}

      {error && !data && (
        <p className="text-danger">
          Analytics backend is unavailable. Retrying every few seconds...
        </p>
      )}

      {data && (
        <>
          {error && (
            <p className="text-[#9a6b00] text-xs mt-0">
              Connection issue - showing last known values.
            </p>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-4">
            {statItems.map((item, idx) => (
              <StatCard key={idx} label={item.label} value={item.value} danger={item.danger} />
            ))}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 mb-4">
            {/* Campaign performance */}
            <div className="border border-border rounded-lg overflow-hidden">
              <div className="px-4 py-3 border-b border-border">
                <h2 className="text-sm font-semibold">Campaign performance</h2>
              </div>
              {campaigns.length === 0 ? (
                <p className="px-4 py-6 text-sm text-text-muted text-center">
                  No campaign data yet.
                </p>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-surface text-text-muted text-left">
                      <th className="px-4 py-2.5 font-medium">Campaign</th>
                      <th className="px-4 py-2.5 font-medium text-right">Clicks</th>
                      <th className="px-4 py-2.5 font-medium text-right">Bots</th>
                      <th className="px-4 py-2.5 font-medium text-right">Saved</th>
                    </tr>
                  </thead>
                  <tbody>
                    {campaigns.map((c) => (
                      <tr key={c.campaign_id} className="border-t border-border">
                        <td className="px-4 py-2.5 font-mono">{c.campaign_id}</td>
                        <td className="px-4 py-2.5 text-right">{formatNumber(c.total_clicks)}</td>
                        <td className="px-4 py-2.5 text-right text-danger">{formatNumber(c.blocked_bots)}</td>
                        <td className="px-4 py-2.5 text-right">{formatMoney(c.saved_money_usd)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>

            {/* Top blocked IPs */}
            <div className="border border-border rounded-lg overflow-hidden">
              <div className="px-4 py-3 border-b border-border">
                <h2 className="text-sm font-semibold">Top blocked IPs</h2>
              </div>
              {topBlockedIPs.length === 0 ? (
                <p className="px-4 py-6 text-sm text-text-muted text-center">
                  No blocked IPs yet.
                </p>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-surface text-text-muted text-left">
                      <th className="px-4 py-2.5 font-medium">IP address</th>
                      <th className="px-4 py-2.5 font-medium text-right">Blocks</th>
                    </tr>
                  </thead>
                  <tbody>
                    {topBlockedIPs.map((row) => (
                      <tr key={row.ip} className="border-t border-border">
                        <td className="px-4 py-2.5 font-mono">{row.ip}</td>
                        <td className="px-4 py-2.5 text-right text-danger">{formatNumber(row.count)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Detection layer breakdown */}
          <div className="border border-border rounded-lg overflow-hidden">
            <div className="px-4 py-3 border-b border-border">
              <h2 className="text-sm font-semibold">Blocks by detection layer</h2>
            </div>
            {reasonEntries.length === 0 ? (
              <p className="px-4 py-6 text-sm text-text-muted text-center">
                No blocked clicks yet.
              </p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-surface text-text-muted text-left">
                    <th className="px-4 py-2.5 font-medium">Reason</th>
                    <th className="px-4 py-2.5 font-medium text-right">Count</th>
                  </tr>
                </thead>
                <tbody>
                  {reasonEntries.map(([reason, count]) => (
                    <tr key={reason} className="border-t border-border">
                      <td className="px-4 py-2.5">{formatReasonLabel(reason)}</td>
                      <td className="px-4 py-2.5 text-right text-danger">{formatNumber(count)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      )}
    </Layout>
  );
}

export default Dashboard;
