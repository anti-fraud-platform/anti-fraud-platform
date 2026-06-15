import { NavLink } from 'react-router-dom';

const navItems = [
  { to: '/', label: 'Dashboard' },
  { to: '/logs', label: 'Logs' },
  { to: '/blacklist', label: 'Blacklist' },
];

function Layout({ title, children }) {
  return (
    <div className="flex min-h-screen font-sans text-text-main">
      {/* Sidebar */}
      <aside className="w-[200px] bg-surface flex-shrink-0 flex flex-col">
        <div className="border-b border-border py-4">
          <div className="text-center text-base font-semibold text-primary">
            AntiFraud
          </div>
        </div>
        <nav className="flex-1 px-3 py-4 flex flex-col gap-1">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `block py-2 rounded-lg text-sm text-center ${
                  isActive
                    ? 'text-primary bg-primary-light'
                    : 'text-text-muted hover:bg-gray-100'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      {/* Main area */}
      <div className="flex-1 flex flex-col bg-white">
        <header className="flex items-center justify-between px-6 py-3 border-b border-border">
          <h1 className="text-base font-semibold">{title}</h1>
          <div className="flex items-center gap-4">
            <span className="bg-success-light text-success text-xs px-2.5 py-1 rounded-lg">
              ● Live
            </span>
            <div className="w-8 h-8 rounded-full bg-primary-light flex items-center justify-center text-xs text-primary font-semibold">
              TD
            </div>
          </div>
        </header>
        <main className="p-6 flex-1">{children}</main>
      </div>
    </div>
  );
}

export default Layout;