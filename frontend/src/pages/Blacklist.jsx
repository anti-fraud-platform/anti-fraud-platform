import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import SkeletonBlacklistRow from '../components/SkeletonBlacklistRow';
const intervalMs = 5000;

function Blacklist() {
  const [loading, setLoading] = useState(true);
  const [blacklistData, setBlacklistData] = useState([]);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    let timer = null;

    async function fetchBlacklist() {
      if (cancelled) return;
      setLoading(true);

      try {
        const response = await fetch('/api/v1/analytics/blacklist/ips');
        if (!response.ok) throw new Error('Failed to fetch');
        const data = await response.json();
        if (cancelled) return;
        setBlacklistData(data.items || []);
        setError(null);
      } catch (err) {
        if (cancelled) return;
        console.error('Error fetching blacklist:', err);
        setError('Failed to load blacklist data');

        timer = setTimeout(fetchBlacklist, intervalMs);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchBlacklist();

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, []);

  const exportToCSV = () => {
    if (blacklistData.length === 0) {
      alert('No data to export');
      return;
    }

    const headers = ['IP', 'Block Count', 'First Blocked', 'Last Blocked'];
    const rows = blacklistData.map(item => [
      item.ip,
      item.block_count,
      item.first_blocked,
      item.last_blocked
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    
    link.setAttribute('href', url);
    link.setAttribute('download', `blacklist_export_${new Date().toISOString().slice(0,10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <Layout title="Blacklist" error={error}>
      <div className="flex justify-between items-center mb-4">
        <div className="text-sm text-text-muted">
          {blacklistData.length > 0 && `Showing ${blacklistData.length} blocked IPs`}
        </div>
        <button
          onClick={exportToCSV}
          disabled={loading || blacklistData.length === 0}
          className="px-4 py-2 bg-primary text-white rounded-lg hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          Export CSV
        </button>
      </div>

      <div className="border border-border rounded-xl overflow-hidden">
        {error && (
          <div className="p-4 text-danger bg-danger-light">{error}</div>
        )}
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="bg-surface text-left text-text-muted">
              <th className="px-3.5 py-2.5 font-medium">IP</th>
              <th className="px-3.5 py-2.5 font-medium">Block Count</th>
              <th className="px-3.5 py-2.5 font-medium">First Blocked</th>
              <th className="px-3.5 py-2.5 font-medium">Last Blocked</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              Array.from({ length: 3 }).map((_, i) => <SkeletonBlacklistRow key={i} />)
            ) : blacklistData.length === 0 ? (
              <tr>
                <td colSpan="4" className="px-3.5 py-4 text-center text-text-muted">
                  No blocked IPs found
                </td>
              </tr>
            ) : (
              blacklistData.map((row, i) => (
                <tr key={i} className="border-t border-border">
                  <td className="px-3.5 py-2.5 font-mono text-text-main">{row.ip}</td>
                  <td className="px-3.5 py-2.5 text-text-main">{row.block_count}</td>
                  <td className="px-3.5 py-2.5 text-text-muted">{row.first_blocked}</td>
                  <td className="px-3.5 py-2.5 text-text-muted">{row.last_blocked}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Layout>
  );
}

export default Blacklist;