// NOTE (for the report): no health-check endpoint exists in the backend,
// so service statuses are static mock values (all "Healthy").
const SERVICES = [
  { name: 'API Gateway', status: 'Healthy' },
  { name: 'Challenge Service', status: 'Healthy' },
  { name: 'Redis', status: 'Healthy' },
  { name: 'PostgreSQL', status: 'Healthy' },
  { name: 'Blacklist Sync', status: 'Healthy' },
];

function SystemHealth() {
  const allHealthy = SERVICES.every((s) => s.status === 'Healthy');

  return (
    <div className="border border-border rounded-lg overflow-hidden flex flex-col flex-1">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">System health</h2>
      </div>

      <div className="p-4 flex items-stretch flex-1">
        {/* Left half: services list */}
        <div className="w-1/2 flex flex-col justify-center gap-3 pr-4">
          {SERVICES.map((s) => (
            <div key={s.name} className="flex items-center justify-between text-xs">
              <span className="text-text-main">{s.name}</span>
              <span className="flex items-center gap-1.5 text-success font-medium">
                <span className="w-1.5 h-1.5 rounded-full bg-success" />
                {s.status}
              </span>
            </div>
          ))}
        </div>

        {/* Right half: shield summary, centered */}
        <div className="w-1/2 flex flex-col items-center justify-center gap-1.5 pl-4 border-l border-border">
          <div
            className="w-16 h-16 rounded-xl flex items-center justify-center"
            style={{ backgroundColor: 'var(--color-primary-light)' }}
          >
            <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="var(--color-primary)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              <path d="M9 12l2 2 4-4" />
            </svg>
          </div>
          <span className="text-xs text-text-muted mt-1">All Systems</span>
          <span className="text-base font-bold text-primary">
            {allHealthy ? 'Operational' : 'Degraded'}
          </span>
        </div>
      </div>
    </div>
  );
}

export default SystemHealth;