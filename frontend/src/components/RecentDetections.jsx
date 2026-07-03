import { useState } from 'react';
import { useLogs } from '../hooks/useLogs';

const LAYER_INFO = {
  suspicious_agent: { n: 1, label: 'User-Agent Check', color: '#8b7cf6' },
  no_js_challenge: { n: 2, label: 'JS Challenge', color: '#38bdf8' },
  challenge_too_fast: { n: 2, label: 'JS Challenge', color: '#38bdf8' },
  challenge_mismatch: { n: 2, label: 'JS Challenge', color: '#38bdf8' },
  suspicious_headers: { n: 3, label: 'Header Analysis', color: '#22d3ee' },
  static_blacklist: { n: 4, label: 'Blacklist', color: '#f0616d' },
  rate_limit_exceeded: { n: 5, label: 'Rate Limiter', color: '#fbbf24' },
};

// Location isn't in the backend data; derive a stable mock country from the IP
// so each IP always shows the same flag (for the redesign only).
const COUNTRIES = [
  { flag: '🇺🇸', name: 'United States' },
  { flag: '🇩🇪', name: 'Germany' },
  { flag: '🇳🇱', name: 'Netherlands' },
  { flag: '🇫🇷', name: 'France' },
  { flag: '🇨🇦', name: 'Canada' },
  { flag: '🇬🇧', name: 'United Kingdom' },
  { flag: '🇯🇵', name: 'Japan' },
];
function countryFor(ip) {
  let sum = 0;
  for (let i = 0; i < (ip || '').length; i++) sum += ip.charCodeAt(i);
  return COUNTRIES[sum % COUNTRIES.length];
}

function timeOf(iso) {
  try {
    return new Date(iso).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return '';
  }
}

function RecentDetections() {
  const [page, setPage] = useState(1);
  const { data } = useLogs({ page, limit: 8 }, 2500);

  const rows = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 1;

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border flex items-center gap-2">
        <h2 className="text-sm font-semibold">Recent detections</h2>
        <span className="text-[10px] text-success font-semibold uppercase tracking-wide">Live</span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-surface text-text-muted text-left text-xs">
              <th className="px-4 py-2.5 font-medium">Time</th>
              <th className="px-4 py-2.5 font-medium">Layer</th>
              <th className="px-4 py-2.5 font-medium">Reason</th>
              <th className="px-4 py-2.5 font-medium">Campaign</th>
              <th className="px-4 py-2.5 font-medium">IP Address</th>
              <th className="px-4 py-2.5 font-medium">Location</th>
              <th className="px-4 py-2.5 font-medium">Status</th>
              <th className="px-4 py-2.5 font-medium">Method</th>
              <th className="px-4 py-2.5 font-medium">User Agent</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td colSpan={9} className="px-4 py-6 text-center text-text-muted">No detections yet.</td>
              </tr>
            ) : (
              rows.map((row) => {
                const layer = LAYER_INFO[row.reason];
                const country = countryFor(row.ip);
                return (
                  <tr key={row.id} className="border-t border-border">
                    <td className="px-4 py-2.5 whitespace-nowrap text-text-muted">{timeOf(row.processed_at)}</td>

                    {/* Layer: allowed -> green Allowed badge; blocked -> layer badge */}
                    <td className="px-4 py-2.5 whitespace-nowrap">
                      {!row.is_bot ? (
                        <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-success-light text-success">Allowed</span>
                      ) : layer ? (
                        <span
                          className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium"
                          style={{ backgroundColor: `${layer.color}22`, color: layer.color }}
                        >
                          <span className="w-3.5 h-3.5 rounded-full flex items-center justify-center text-[8px] text-white" style={{ backgroundColor: layer.color }}>{layer.n}</span>
                          {layer.label}
                        </span>
                      ) : (
                        <span className="text-text-muted text-xs">—</span>
                      )}
                    </td>

                    {/* Reason: allowed -> dash */}
                    <td className="px-4 py-2.5 whitespace-nowrap font-mono text-xs text-text-muted">
                      {!row.is_bot ? '—' : (row.reason || '—')}
                    </td>

                    <td className="px-4 py-2.5 whitespace-nowrap font-mono text-xs">{row.campaign_id}</td>
                    <td className="px-4 py-2.5 whitespace-nowrap font-mono text-xs">{row.ip}</td>

                    {/* Location (mock) */}
                    <td className="px-4 py-2.5 whitespace-nowrap text-xs">
                      <span className="mr-1.5">{country.flag}</span>{country.name}
                    </td>

                    <td className="px-4 py-2.5 whitespace-nowrap">
                      {row.is_bot ? (
                        <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-danger-light text-danger">Blocked</span>
                      ) : (
                        <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-success-light text-success">Allowed</span>
                      )}
                    </td>

                    <td className="px-4 py-2.5 whitespace-nowrap font-mono text-xs text-text-muted">GET</td>

                    <td className="px-4 py-2.5 max-w-[220px] truncate text-xs text-text-muted" title={row.user_agent}>{row.user_agent}</td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-text-muted">
        <span>Showing page {page} of {totalPages} ({total} results)</span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="px-3 py-1 rounded border border-border disabled:opacity-40 hover:bg-surface transition-colors"
          >
            Prev
          </button>
          <button
            type="button"
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
            className="px-3 py-1 rounded border border-border disabled:opacity-40 hover:bg-surface transition-colors"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
}

export default RecentDetections;