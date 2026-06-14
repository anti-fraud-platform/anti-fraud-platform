import { NavLink } from 'react-router-dom'

// Reusable admin shell: sidebar + top bar wrap any page passed as children.
const navItems = [
  { to: '/', label: 'Dashboard' },
  { to: '/logs', label: 'Logs' },
  { to: '/blacklist', label: 'Blacklist' },
]

function Layout({ title, children }) {
  return (
    <div style={{ display: 'flex', minHeight: '100vh', fontFamily: 'sans-serif', color: '#1a1a1a' }}>
      <aside style={{ width: 200, background: '#f4f5f7', padding: '1rem 0.75rem', flexShrink: 0 }}>
        <div style={{ padding: '0 8px 16px', borderBottom: '1px solid #e2e4e8', marginBottom: 12 }}>
          <span style={{ fontSize: 16, fontWeight: 600, color: '#185fa5' }}>AntiFraud</span>
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              style={({ isActive }) => ({
                padding: '9px 10px',
                borderRadius: 8,
                fontSize: 14,
                textDecoration: 'none',
                color: isActive ? '#185fa5' : '#5f5e5a',
                background: isActive ? '#e6f1fb' : 'transparent',
              })}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: '#fff' }}>
        <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 24px', borderBottom: '1px solid #e2e4e8' }}>
          <span style={{ fontSize: 16, fontWeight: 600 }}>{title}</span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
            <span style={{ background: '#e1f5ee', color: '#0f6e56', fontSize: 12, padding: '4px 10px', borderRadius: 8 }}>
              ● Live
            </span>
            <div style={{ width: 30, height: 30, borderRadius: '50%', background: '#e6f1fb', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, color: '#185fa5' }}>TD</div>
          </div>
        </header>

        <main style={{ padding: 24, flex: 1 }}>{children}</main>
      </div>
    </div>
  )
}

export default Layout