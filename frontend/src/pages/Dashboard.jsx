import Layout from '../components/Layout'
import StatCard from '../components/StatCard'
import { useStats } from '../hooks/useStats'

// Formats a number with thousands separators, e.g. 12500 -> "12,500".
function formatNumber(n) {
  return Number(n).toLocaleString('en-US')
}

// Formats money, e.g. 24900 -> "$24,900".
function formatMoney(n) {
  return '$' + Number(n).toLocaleString('en-US', { maximumFractionDigits: 0 })
}

function Dashboard() {
  // All state + polling live in the hook. Dashboard just reads the result.
  const { data, loading, error } = useStats(2500)

  return (
    <Layout title="Dashboard">
      {/* First load, no data yet */}
      {loading && !data && <p style={{ color: '#5f5e5a' }}>Loading stats...</p>}

      {/* Error, and we never managed to load anything */}
      {error && !data && (
        <p style={{ color: '#a32d2d' }}>
          Analytics backend is unavailable. Retrying every few seconds...
        </p>
      )}

      {/* We have data - render cards. Values passed down as props. */}
      {data && (
        <>
          {/* Small notice if the latest poll failed but we still show last good data */}
          {error && (
            <p style={{ color: '#9a6b00', fontSize: 13, marginTop: 0 }}>
              Connection issue - showing last known values.
            </p>
          )}

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12, marginBottom: 16 }}>
            <StatCard label="Total clicks" value={formatNumber(data.total_clicks)} />
            <StatCard label="Blocked bots" value={formatNumber(data.blocked_bots)} danger />
            <StatCard label="Money saved" value={formatMoney(data.saved_money_usd)} />
          </div>

          {/* Chart placeholders (next week) */}
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
                {label} - chart (next week)
              </div>
            ))}
          </div>
        </>
      )}
    </Layout>
  )
}

export default Dashboard