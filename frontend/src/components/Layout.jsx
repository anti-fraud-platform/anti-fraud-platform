import { NavLink } from 'react-router-dom';
import { useTheme } from '../hooks/useTheme';

const navItems = [
  { to: '/', label: 'Dashboard' },
  { to: '/logs', label: 'Logs' },
  { to: '/blacklist', label: 'Blacklist' },
];

function Layout({ title, children }) {
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="flex min-h-screen font-sans text-text-main">
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
                    : 'text-text-muted hover:bg-primary-light'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className="flex-1 flex flex-col bg-app-bg">
        <header className="flex items-center justify-between px-6 py-3 border-b border-border">
          <h1 className="text-base font-semibold">{title}</h1>
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={toggleTheme}
              aria-label="Toggle theme"
              className="w-8 h-8 rounded-lg border border-border flex items-center justify-center text-text-muted hover:bg-surface transition-colors"
            >
              {theme === 'dark' ? (
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="4" />
                  <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
                </svg>
              ) : (
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
                </svg>
              )}
            </button>

            <span className="bg-success-light text-success text-xs px-2.5 py-1 rounded-lg">
              Live
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