import { Link } from 'react-router-dom';
import { useLogs } from '../hooks/useLogs';

const LAYER_LABEL = {
  suspicious_agent: { prefix: 'Suspicious user agent detected from IP', layer: '', color: '#8b7cf6', ipFirst: false },
  no_js_challenge: { prefix: 'blocked by', layer: 'JS Challenge', color: '#38bdf8', ipFirst: true },
  challenge_too_fast: { prefix: 'blocked by', layer: 'JS Challenge (too fast)', color: '#38bdf8', ipFirst: true },
  challenge_mismatch: { prefix: 'blocked by', layer: 'JS Challenge (mismatch)', color: '#38bdf8', ipFirst: true },
  suspicious_headers: { prefix: 'blocked by', layer: 'Header Analysis', color: '#22d3ee', ipFirst: true },
  static_blacklist: { prefix: 'blocked by', layer: 'Static Blacklist', color: '#f0616d', ipFirst: true },
  rate_limit_exceeded: { prefix: 'Rate limit exceeded for IP', layer: '', color: '#fbbf24', ipFirst: false },
};

function timeOf(iso) {
  try {
    return new Date(iso).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return '';
  }
}

function RecentActivity() {
  const { data } = useLogs({ page: 1, limit: 12 }, 2500);
  const rows = (data?.data ?? []).filter((r) => r.is_bot).slice(0, 5);

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Recent activity</h2>
      </div>

      {rows.length === 0 ? (
        <p className="px-4 py-6 text-sm text-text-muted text-center">No recent activity.</p>
      ) : (
        <>
          <div className="p-4 flex flex-col gap-3">
            {rows.map((r) => {
              const info = LAYER_LABEL[r.reason] || { prefix: 'blocked', layer: '', color: 'var(--color-text-muted)', ipFirst: true };
              return (
                <div key={r.id} className="flex items-start gap-2 text-xs">
                  <span className="w-1.5 h-1.5 rounded-full mt-1.5 flex-shrink-0" style={{ backgroundColor: info.color }} />
                  <div className="flex-1 min-w-0">
                    <span className="text-text-muted font-mono mr-2">{timeOf(r.processed_at)}</span>
                    {info.ipFirst ? (
                      <>
                        <span className="text-text-main">IP </span>
                        <span className="font-mono text-text-main">{r.ip}</span>{' '}
                        <span className="text-text-muted">{info.prefix} </span>
                        <span style={{ color: info.color }} className="font-medium">{info.layer}</span>
                      </>
                    ) : (
                      <>
                        <span className="text-text-muted">{info.prefix} </span>
                        <span className="font-mono text-text-main">{r.ip}</span>
                      </>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          <Link
            to="/logs"
            className="w-full py-2.5 text-xs font-medium text-primary border-t border-border hover:bg-primary-light transition-colors flex items-center justify-center gap-1"
          >
            View all activity
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M13 6l6 6-6 6" />
            </svg>
          </Link>
        </>
      )}
    </div>
  );
}

export default RecentActivity;