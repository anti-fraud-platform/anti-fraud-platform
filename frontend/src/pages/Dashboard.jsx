import Layout from '../components/Layout'
import { stats } from '../data/mockData'

// Dashboard page: stat cards + empty chart slots (charts come next week).
function Dashboard() {
  return (
    <Layout title="Dashboard">
      {/* Stat cards */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12, marginBottom: 16 }}>
        {stats.map((s) => (
          <div key={s.id} style={{ background: '#f4f5f7', borderRadius: 8, padding: 16 }}>
            <p style={{ fontSize: 13, color: '#5f5e5a', margin: '0 0 6px' }}>{s.label}</p>
            <p style={{ fontSize: 24, fontWeight: 600, margin: 0, color: s.danger ? '#a32d2d' : '#1a1a1a' }}>
              {s.value}
            </p>
          </div>
        ))}
      </div>

      {/* Chart placeholders */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        {['Traffic over time', 'Bot ratio'].map((label) => (
          <div
            key={label}
            style={{
              border: '1px dashed #c9ccd2',
              borderRadius: 12,
              height: 220,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#9a9a96',
              fontSize: 13,
            }}
          >
            {label} — chart (Week 2)
          </div>
        ))}
      </div>
    </Layout>
  )
}

export default Dashboard