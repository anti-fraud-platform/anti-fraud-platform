import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import SkeletonBlacklistRow from '../components/SkeletonBlacklistRow';
import { fetchBlacklistIps } from '../api/analytics';

const intervalMs = 5000;

const SOURCE_LABELS = {
  dynamic_blacklist: 'Auto-blacklist',
  geoip_policy: 'GeoIP / ASN policy',
};

function formatSource(source) {
  if (!source) return '—';
  return source
    .split(',')
    .map((s) => SOURCE_LABELS[s] || s)
    .join(' + ');
}

function csvEscape(value) {
  const str = String(value ?? '');
  if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
    return '"' + str.replace(/"/g, '""') + '"';
  }
  return str;
}

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
        const data = await fetchBlacklistIps();
        if (cancelled) return;
        setBlacklistData(data.items || []);
        setError(null);
      } catch (err) {
        if (cancelled) return;
        console.error('Error fetching blocked IPs:', err);
        setError('Failed to load blocked IP data');
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchBlacklist();
    timer = setInterval(fetchBlacklist, intervalMs);

    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, []);

  const exportToCSV = () => {
    if (blacklistData.length === 0) {
      alert('No data to export');
      return;
    }

    const headers = ['IP', 'Source', 'Block Count', 'First Blocked', 'Last Blocked'];
    const rows = blacklistData.map(item => [
      item.ip,
      csvEscape(formatSource(item.source)),
      item.block_count,
      csvEscape(item.first_blocked),
      csvEscape(item.last_blocked)
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    
    link.setAttribute('href', url);
    link.setAttribute('download', `blocked_ips_export_${new Date().toISOString().slice(0,10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <Layout title="Blocked IPs" error={error}>
      <div className="flex justify-between items-center mb-4">
        <div className="text-sm text-text-muted">
          {blacklistData.length > 0 && `Showing ${blacklistData.length} blocked IPs (GeoIP policy + auto-blacklist)`}
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
              <th className="px-3.5 py-2.5 font-medium">Source</th>
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
                <td colSpan="5" className="px-3.5 py-4 text-center text-text-muted">
                  No blocked IPs found
                </td>
              </tr>
            ) : (
              blacklistData.map((row, i) => (
                <tr key={i} className="border-t border-border">
                  <td className="px-3.5 py-2.5 font-mono text-text-main">{row.ip}</td>
                  <td className="px-3.5 py-2.5 text-text-main">{formatSource(row.source)}</td>
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
