function StatCard({ label, value, danger }) {
  return (
    <div className="bg-surface rounded-lg p-4 text-center">
      <p className="text-xs text-text-muted mb-1.5">{label}</p>
      <p className={`text-2xl font-semibold ${danger ? 'text-danger' : 'text-text-main'}`}>
        {value}
      </p>
    </div>
  );
}

export default StatCard;