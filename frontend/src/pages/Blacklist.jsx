import Layout from '../components/Layout'
import { blacklist } from '../data/mockData'

// Blacklist page: table of blocked IPs (mock data for now).
function Blacklist() {
  return (
    <Layout title="Blacklist">
      <div style={{ border: '1px solid #e2e4e8', borderRadius: 12, overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
          <thead>
            <tr style={{ background: '#f4f5f7', textAlign: 'left', color: '#5f5e5a' }}>
              <th style={{ padding: '10px 14px', fontWeight: 500 }}>IP</th>
              <th style={{ padding: '10px 14px', fontWeight: 500 }}>Reason</th>
              <th style={{ padding: '10px 14px', fontWeight: 500 }}>Blocked at</th>
            </tr>
          </thead>
          <tbody>
            {blacklist.map((row, i) => (
              <tr key={i} style={{ borderTop: '1px solid #e2e4e8' }}>
                <td style={{ padding: '10px 14px', fontFamily: 'monospace' }}>{row.ip}</td>
                <td style={{ padding: '10px 14px', color: '#5f5e5a' }}>{row.reason}</td>
                <td style={{ padding: '10px 14px', color: '#5f5e5a' }}>{row.blockedAt}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Layout>
  )
}

export default Blacklist