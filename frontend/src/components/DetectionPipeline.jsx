const JS_REASONS = ['no_js_challenge', 'challenge_too_fast', 'challenge_mismatch'];

function DetectionPipeline({ data }) {
  const rb = data?.reason_breakdown ?? {};
  const total = data?.total_clicks ?? 0;
  const allowed = data?.allowed_count ?? 0;

  const jsBlocked = JS_REASONS.reduce((sum, r) => sum + (rb[r] || 0), 0);

  const stages = [
    { label: 'Incoming', badge: null, reasons: [], value: total, valueClass: 'text-text-main', accent: 'var(--color-text-muted)', pctClass: 'text-text-muted' },
    { label: 'User-Agent Check', badge: 1, reasons: ['suspicious_agent'], value: rb['suspicious_agent'] || 0, valueClass: 'text-text-main', accent: '#8b7cf6', pctClass: 'text-[#8b7cf6]' },
    { label: 'JS Challenge', badge: 2, reasons: JS_REASONS, value: jsBlocked, valueClass: 'text-text-main', accent: '#38bdf8', pctClass: 'text-[#38bdf8]' },
    { label: 'Header / Fingerprint', badge: 3, reasons: ['suspicious_headers'], value: rb['suspicious_headers'] || 0, valueClass: 'text-text-main', accent: '#22d3ee', pctClass: 'text-[#22d3ee]' },
    { label: 'GeoIP / ASN Policy', badge: 4, reasons: ['geoip_policy'], value: rb['geoip_policy'] || 0, valueClass: 'text-danger', accent: '#f0616d', pctClass: 'text-danger' },
    { label: 'Rate Limiter', badge: 5, reasons: ['rate_limit_exceeded'], value: rb['rate_limit_exceeded'] || 0, valueClass: 'text-text-main', accent: '#fbbf24', pctClass: 'text-[#fbbf24]' },
    { label: 'Allowed', badge: null, reasons: [], value: allowed, valueClass: 'text-success', accent: '#34d399', pctClass: 'text-success' },
  ];

  const legend = [
    { n: 1, label: 'User-Agent', color: '#8b7cf6' },
    { n: 2, label: 'JS Challenge', color: '#38bdf8' },
    { n: 3, label: 'Header Analysis', color: '#22d3ee' },
    { n: 4, label: 'GeoIP / ASN', color: '#f0616d' },
    { n: 5, label: 'Rate Limiter', color: '#fbbf24' },
    { n: null, label: 'Allowed', color: '#34d399' },
  ];

  const pct = (v) => (total > 0 ? ((v / total) * 100).toFixed(1) : '0.0');

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Detection pipeline (flow)</h2>
      </div>

      <div className="px-5 pb-5 pt-9">
        <div className="flex items-stretch gap-1">
          {stages.map((s, i) => {
            const isEndpoint = s.reasons.length === 0; // Incoming / Allowed
            return (
              <div key={s.label} className="flex items-stretch gap-1 flex-1 min-w-0">
                <div
                  className={`relative flex-1 min-w-0 h-[190px] border border-border rounded-xl px-2 py-4 flex flex-col items-center bg-app-bg ${
                    isEndpoint ? 'justify-center gap-3' : ''
                  }`}
                  style={{ borderTopColor: s.accent, borderTopWidth: '3px' }}
                >
                  {s.badge !== null && (
                    <span
                      className="absolute -top-4 left-1/2 -translate-x-1/2 w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold text-white shadow-md border-2 border-app-bg z-20"
                      style={{ backgroundColor: s.accent }}
                    >
                      {s.badge}
                    </span>
                  )}

                  {isEndpoint ? (
                    /* Incoming / Allowed: title + value grouped in the vertical center */
                    <>
                      <span className="block w-full text-center text-[11px] text-text-muted uppercase tracking-wide font-semibold leading-tight">
                        {s.label}
                      </span>
                      <div className="flex flex-col items-center">
                        <span className={`text-2xl font-bold leading-none ${s.valueClass}`}>
                          {Number(s.value).toLocaleString('en-US')}
                        </span>
                        <span className={`text-xs mt-1.5 font-semibold ${s.pctClass}`}>
                          {pct(s.value)}%
                        </span>
                      </div>
                    </>
                  ) : (
                    /* Detection layers: title top, reasons middle, value bottom */
                    <>
                      <div className="flex items-start justify-center min-h-[32px] w-full">
                        <span className="block w-full text-center text-[11px] text-text-muted uppercase tracking-wide font-semibold leading-tight">
                          {s.label}
                        </span>
                      </div>
                      <div className="flex-1 flex flex-col items-center justify-center gap-0.5 w-full">
                        {s.reasons.map((r) => (
                          <span key={r} className="text-[9px] text-text-muted font-mono leading-tight text-center">
                            {r}
                          </span>
                        ))}
                      </div>
                      <div className="flex flex-col items-center">
                        <span className={`text-2xl font-bold leading-none ${s.valueClass}`}>
                          {Number(s.value).toLocaleString('en-US')}
                        </span>
                        <span className={`text-xs mt-1.5 font-semibold ${s.pctClass}`}>
                          {pct(s.value)}%
                        </span>
                      </div>
                    </>
                  )}
                </div>

                {/* Animated dot pyramid between stages */}
                {i < stages.length - 1 && (
                  <div
                    className="flex items-center flex-shrink-0 pyramid-flow"
                    style={{ color: stages[i + 1].accent }}
                  >
                    {/* column of 3 */}
                    <span className="flex flex-col gap-1">
                      <span className="w-1 h-1 rounded-full bg-current" />
                      <span className="w-1 h-1 rounded-full bg-current" />
                      <span className="w-1 h-1 rounded-full bg-current" />
                    </span>
                    {/* column of 2 */}
                    <span className="flex flex-col gap-1 ml-1">
                      <span className="w-1 h-1 rounded-full bg-current" />
                      <span className="w-1 h-1 rounded-full bg-current" />
                    </span>
                    {/* column of 1 (tip) */}
                    <span className="flex flex-col ml-1">
                      <span className="w-1 h-1 rounded-full bg-current" />
                    </span>
                  </div>
                )}
              </div>
            );
          })}
        </div>

        {/* Legend */}
        <div className="flex flex-wrap items-center justify-center gap-4 mt-4 pt-4 border-t border-border">
          {legend.map((l) => (
            <div key={l.label} className="flex items-center gap-1.5 text-[11px] text-text-muted">
              <span
                className="w-4 h-4 rounded-full flex items-center justify-center text-[9px] font-semibold text-white"
                style={{ backgroundColor: l.color }}
              >
                {l.n ?? ''}
              </span>
              {l.label}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default DetectionPipeline;
