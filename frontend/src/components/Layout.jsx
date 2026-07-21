import { House, Logs, ShieldBan, Shield, Fingerprint, Search, LogOut } from 'lucide-react';
import { NavLink, useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { getMe } from '../api/auth';

const navItems = [
  { to: '/analytics', label: 'Dashboard', icon: <House /> },
  { to: '/logs', label: 'Logs', icon: <Logs /> },
  { to: '/blacklist', label: 'Blacklist', icon: <ShieldBan /> },
];

function Layout({ title, error, children }) {
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const [user, setUser] = useState(null);
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) return;

    getMe()
      .then((data) => setUser({ username: data.username, role: data.role }))
      .catch(() => setUser(null));
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('token');
    navigate('/login', { replace: true });
  };

  return (
    <div className="flex min-h-screen font-sans text-text-main">
      <aside className="w-[200px] bg-surface">
        <div className="sticky top-0 h-screen flex-shrink-0 flex flex-col">
          <div className="border-b border-border py-4">
            <div className="flex flex-col items-center text-center">
              <div style={{color: "#8b7cf6"}}>
                <Shield size={40}>
                  <Fingerprint size={10} x={7} y={7} />
                  <Search size={12} x={12} y={12} />
                </Shield>
              </div>
              <h1 className="text-base font-semibold">ANTIFRAUD</h1>
              <p className="text-xs text-text-muted">Click Fraud Protection</p>
            </div>
          </div>
          <nav className="flex-1 px-3 py-4 flex flex-col gap-1">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/analytics'}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-4 py-2 rounded-lg text-sm ${
                    isActive
                      ? 'text-primary bg-primary-light'
                      : 'text-text-muted hover:bg-primary-light'
                  }`
                }
              >
                {item.icon}
                <div>{item.label}</div>
              </NavLink>
            ))}
          </nav>

          <div
            className="border-t border-border"
            onMouseEnter={() => setMenuOpen(true)}
            onMouseLeave={() => setMenuOpen(false)}
          >
            <button
              type="button"
              className="w-full px-3 py-3 flex flex-row items-center gap-1.5 justify-between hover:bg-surface transition-colors"
            >
              <div className="flex flex-row gap-3 items-center">
                <div className="w-8 h-8 bg-primary-light flex rounded-full items-center justify-center text-xs text-primary font-semibold">
                  {user ? user.username.slice(0, 2).toUpperCase() : '??'}
                </div>
                <div className="flex flex-col text-left">
                  <div className="text-sm font-semibold text-text-main">
                    {user ? user.username : 'Loading...'}
                  </div>
                  <div className="text-xs text-text-muted">
                    {user ? user.role : ''}
                  </div>
                </div>
              </div>
              <svg
                className={`block flex-shrink-0 ${menuOpen ? 'rotate-180' : ''}`}
                width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
              >
                <polyline points="6 9 12 15 18 9" />
              </svg>
            </button>

            <div
              className={`border-t border-border px-3 transition-all duration-200 ${
                menuOpen
                  ? 'opacity-100 max-h-20 py-2'
                  : 'opacity-0 max-h-0 overflow-hidden py-0'
              }`}
            >
              <button
                onClick={handleLogout}
                className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-text-main hover:bg-primary-light hover:text-primary transition-colors cursor-pointer"
              >
                <LogOut size={16} />
                Sign out
              </button>
            </div>
          </div>

          <div className="border-t px-4 py-4 border-border flex flex-col text-xs text-primary gap-1">
              <div>@ 2026 AntiFraud</div>
              <div>v1.0.0</div>
          </div>
        </div>
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

            <div className="flex items-center gap-1.5 px-3 py-1.5 h-9 rounded-lg border border-border text-sm text-text-muted leading-5">
              <span>Last 7 days</span>
              <svg className="block flex-shrink-0" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="6 9 12 15 18 9" />
              </svg>
            </div>

            <span
              className={`text-xs px-2.5 py-1 rounded-lg ${
                error
                  ? 'bg-danger-light text-danger'
                  : 'bg-success-light text-success'
              }`}
            >
              {error ? 'Error' : 'Live'}
            </span>
          </div>
        </header>
        <main className="p-6 flex-1">{children}</main>
      </div>
    </div>
  );
}

export default Layout;