// Presentational card. Receives everything via props - no own state.
function StatCard({ label, value, danger }) {
  return (
    <div style={{ background: '#f4f5f7', borderRadius: 8, padding: 16 }}>
      <p style={{ fontSize: 13, color: '#5f5e5a', margin: '0 0 6px' }}>{label}</p>
      <p style={{ fontSize: 24, fontWeight: 600, margin: 0, color: danger ? '#a32d2d' : '#1a1a1a' }}>
        {value}
      </p>
    </div>
  )
}

export default StatCard