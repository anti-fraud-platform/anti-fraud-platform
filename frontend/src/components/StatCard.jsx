function StatCard({ label, value, danger, icon, delta, deltaUp }) {
  return (
    <div className="bg-surface rounded-lg p-4 flex items-start gap-3">
      {icon && (
        <div className="w-9 h-9 rounded-lg bg-primary-light flex items-center justify-center text-primary flex-shrink-0">
          {icon}
        </div>
      )}
      <div className="flex-1 min-w-0">
        <p className="text-xs text-text-muted mb-1">{label}</p>
        <p className={`text-2xl font-bold leading-none ${danger ? 'text-danger' : 'text-text-main'}`}>
          {value}
        </p>
        {delta && (
          <p className={`text-[11px] mt-1.5 font-medium ${deltaUp ? 'text-success' : 'text-danger'}`}>
            {deltaUp ? '▲' : '▼'} {delta} <span className="text-text-muted font-normal">vs prev 7 days</span>
          </p>
        )}
      </div>
    </div>
  );
}

export default StatCard;