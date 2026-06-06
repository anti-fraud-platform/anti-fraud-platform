import Layout from '../components/Layout'
import { recentClicks } from '../data/mockData'

// Logs page: table of recent clicks (mock data for now).
function Logs() {
  return (
    <Layout title="Logs">
      <div style={{ border: '1px solid #e2e4e8', borderRadius: 12, overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
          <thead>
            <tr style={{ background: '#f4f5f7', textAlign: 'left', color: '#5f5e5a' }}>
              <th style={{ padding: '10px 14px', fontWeight: 500 }}>IP</th>
              <th style={{ padding: '10px 14px', fontWeight: 500 }}>User-Agent</th>
              <th style={{ padding: '10px 14px', fontWeight: 500 }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {recentClicks.map((row, i) => (
              <tr key={i} style={{ borderTop: '1px solid #e2e4e8' }}>
                <td style={{ padding: '10px 14px', fontFamily: 'monospace' }}>{row.ip}</td>
                <td style={{ padding: '10px 14px', color: '#5f5e5a' }}>{row.agent}</td>
                <td style={{ padding: '10px 14px' }}>
                  <span
                    style={{
                      padding: '2px 10px',
                      borderRadius: 8,
                      fontSize: 12,
                      background: row.status === 'bot' ? '#fcebeb' : '#e1f5ee',
                      color: row.status === 'bot' ? '#a32d2d' : '#0f6e56',
                    }}
                  >
                    {row.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Layout>
  )
}

export default Logs