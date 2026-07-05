import { useState } from 'react';
import Layout from '../components/Layout';
import { useLogs } from '../hooks/useLogs';

function formatDate(iso) {
  try {
    return new Date(iso).toLocaleString('en-US', {
      month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
  } catch {
    return iso;
  }
}

function formatReason(reason, isBot) {
  if (!isBot) return 'Allowed';
  const map = {
    suspicious_agent: 'Suspicious UA',
    no_js_challenge: 'No JS challenge',
    challenge_too_fast: 'Challenge too fast',
    challenge_mismatch: 'Challenge mismatch',
    suspicious_headers: 'Suspicious headers',
    static_blacklist: 'Static blacklist',
    rate_limit_exceeded: 'Rate limit exceeded',
  };
  return map[reason] || reason || '—';
}

// Row tint by type — semi-transparent so it works in light AND dark themes.
function rowTint(isBot, reason) {
  if (!isBot) return 'transparent';
  if (reason === 'suspicious_agent') return 'rgba(240, 97, 109, 0.08)';   // red-ish
  if (reason === 'static_blacklist') return 'rgba(240, 97, 109, 0.12)';
  return 'rgba(139, 124, 246, 0.08)'; // purple-ish for other bot reasons
}

function Logs() {
  const [page, setPage] = useState(1);
  const [campaign, setCampaign] = useState('');
  const [type, setType] = useState('');
  const [reason, setReason] = useState('');
  const [jumpValue, setJumpValue] = useState('');

  const params = { page, limit: 20 };
  if (campaign) params.campaign_id = campaign;
  if (type === 'bots') params.is_bot = true;
  if (type === 'clean') params.is_bot = false;
  if (reason) params.reason = reason;

  const { data, loading, error } = useLogs(params, 2500);

  const logs = data?.data ?? [];
  const totalPages = data?.total_pages ?? 1;

  return (
    <Layout title="Logs" error={error}>
      {/* Filters */}
      <div className="flex flex-wrap gap-4 mb-4">
        <div className="flex flex-col gap-1">
          <label className="text-xs text-text-muted">Campaign</label>
          <input
            type="text"
            value={campaign}
            onChange={(e) => { setCampaign(e.target.value); setPage(1); }}
            placeholder="All campaigns"
            className="px-3 py-2 rounded-lg border border-border bg-app-bg text-text-main text-sm w-48"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs text-text-muted">Type</label>
          <select
            value={type}
            onChange={(e) => { setType(e.target.value); setPage(1); }}
            className="px-3 py-2 rounded-lg border border-border bg-app-bg text-text-main text-sm"
          >
            <option value="">All traffic</option>
            <option value="bots">Bots only</option>
            <option value="clean">Clean only</option>
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs text-text-muted">Reason</label>
          <select
            value={reason}
            onChange={(e) => { setReason(e.target.value); setPage(1); }}
            className="px-3 py-2 rounded-lg border border-border bg-app-bg text-text-main text-sm"
          >
            <option value="">All reasons</option>
            <option value="suspicious_agent">Suspicious UA</option>
            <option value="no_js_challenge">No JS challenge</option>
            <option value="challenge_too_fast">Challenge too fast</option>
            <option value="challenge_mismatch">Challenge mismatch</option>
            <option value="suspicious_headers">Suspicious headers</option>
            <option value="static_blacklist">Static blacklist</option>
            <option value="rate_limit_exceeded">Rate limit exceeded</option>
          </select>
        </div>
      </div>

      {error && !data && (
        <p className="text-danger mb-3">Analytics backend is unavailable. Retrying...</p>
      )}

      {/* Table */}
      <div className="border border-border rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-surface text-text-muted text-left text-xs">
                <th className="px-4 py-3 font-medium">IP</th>
                <th className="px-4 py-3 font-medium">Campaign</th>
                <th className="px-4 py-3 font-medium">User-Agent</th>
                <th className="px-4 py-3 font-medium">Reason</th>
                <th className="px-4 py-3 font-medium">Date/Time</th>
              </tr>
            </thead>
            <tbody>
              {loading && !data ? (
                <tr><td colSpan={5} className="px-4 py-6 text-center text-text-muted">Loading...</td></tr>
              ) : logs.length === 0 ? (
                <tr><td colSpan={5} className="px-4 py-6 text-center text-text-muted">No logs found.</td></tr>
              ) : (
                logs.map((log) => (
                  <tr
                    key={log.id}
                    className="border-t border-border"
                    style={{ backgroundColor: rowTint(log.is_bot, log.reason) }}
                  >
                    <td className="px-4 py-3 font-mono text-xs text-text-main">{log.ip}</td>
                    <td className="px-4 py-3 text-text-main">{log.campaign_id}</td>
                    <td className="px-4 py-3 max-w-[380px] truncate text-text-muted" title={log.user_agent}>{log.user_agent}</td>
                    <td className="px-4 py-3 whitespace-nowrap text-text-main">{formatReason(log.reason, log.is_bot)}</td>
                    <td className="px-4 py-3 whitespace-nowrap text-text-muted">{formatDate(log.processed_at)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center gap-3 px-4 py-3 border-t border-border text-xs text-text-muted">
          <button
            type="button"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="px-3 py-1 rounded border border-border disabled:opacity-40 hover:bg-surface transition-colors"
          >
            Previous
          </button>
          <span>Page {page} of {totalPages}</span>
          <input
            type="text"
            value={jumpValue}
            onChange={(e) => setJumpValue(e.target.value)}
            placeholder="Jump to..."
            className="px-2 py-1 rounded border border-border bg-app-bg text-text-main w-20"
          />
          <button
            type="button"
            onClick={() => {
              const n = parseInt(jumpValue, 10);
              if (!isNaN(n) && n >= 1 && n <= totalPages) { setPage(n); setJumpValue(''); }
            }}
            className="px-3 py-1 rounded border border-border hover:bg-surface transition-colors"
          >
            Go
          </button>
          <button
            type="button"
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
            className="px-3 py-1 rounded border border-border disabled:opacity-40 hover:bg-surface transition-colors ml-auto"
          >
            Next
          </button>
        </div>
      </div>
    </Layout>
  );
}

export default Logs;