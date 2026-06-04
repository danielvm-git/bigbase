import { useEffect, useState, type ReactNode } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { Icon, type IconName } from './components/Icon'
import { ThemePicker } from './components/ThemePicker'
import { useTheme } from './hooks/useTheme'

interface UserInfo { id: number; email: string }

interface NavItem {
  to: string
  label: string
  icon: IconName
  end?: boolean
}

function NavSection({ title, items }: { title: string; items: NavItem[] }) {
  return (
    <div className="sidebar-section">
      <div className="sidebar-section-title">{title}</div>
      <ul className="sidebar-nav">
        {items.map(item => (
          <li key={item.to}>
            <NavLink to={item.to} end={item.end}>
              <Icon name={item.icon} size={18} />
              <span>{item.label}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  )
}

export default function Layout() {
  const nav = useNavigate()
  const { theme, toggle: toggleTheme, accent, setAccent } = useTheme()
  const [user, setUser] = useState<UserInfo | null>(null)
  const [appVersion, setAppVersion] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(false)

  useEffect(() => {
    fetch('/api/auth/me')
      .then(r => {
        if (!r.ok) throw new Error('unauthorized')
        return r.json() as Promise<UserInfo>
      })
      .then(setUser)
      .catch(() => nav('/login'))
  }, [nav])

  useEffect(() => {
    fetch('/api/version')
      .then(r => r.json())
      .then(d => setAppVersion(d.version))
      .catch(() => {})
  }, [])

  const handleLogout = () => nav('/login')

  const initial = user?.email?.[0]?.toUpperCase() || '?'

  const buildNav: NavItem[] = [
    { to: '/deploy', label: 'Sites', icon: 'rocket' },
    { to: '/functions', label: 'Functions', icon: 'box' },
  ]
  const dataNav: NavItem[] = [
    { to: '/data', label: 'Data Studio', icon: 'database' },
    { to: '/sql', label: 'SQL Editor', icon: 'terminal' },
    { to: '/storage', label: 'Storage', icon: 'hard-drive' },
  ]
  const authNav: NavItem[] = [{ to: '/users', label: 'Users', icon: 'users' }]
  const engageNav: NavItem[] = [{ to: '/messaging', label: 'Messaging', icon: 'mail' }]
  const devOpsNav: NavItem[] = [
    { to: '/repos', label: 'Git Repos', icon: 'git-branch' },
    { to: '/cici', label: 'CI/CD', icon: 'git-pull-request' },
    { to: '/monitoring', label: 'Monitoring', icon: 'activity' },
    { to: '/forge', label: 'Forge', icon: 'hammer' },
    { to: '/realtime', label: 'Realtime', icon: 'radio' },
  ]

  let footerBlock: ReactNode = null
  if (user) {
    footerBlock = (
      <div className="sidebar-footer">
        <div className="sidebar-section-title sidebar-appearance-label">Appearance</div>
        <div className="sidebar-appearance">
          <button
            type="button"
            onClick={toggleTheme}
            className="btn btn-secondary btn-sm sidebar-appearance-btn"
            title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
            aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
          >
            <Icon name={theme === 'light' ? 'moon' : 'sun'} size={16} />
            <span>{theme === 'light' ? 'Dark mode' : 'Light mode'}</span>
          </button>
          <label className="sidebar-accent-label">
            <span className="dim">Accent</span>
            <ThemePicker value={accent} onChange={setAccent} />
          </label>
        </div>
        <ul className="sidebar-nav sidebar-footer-nav">
          <li>
            <NavLink to="/settings">
              <Icon name="settings" size={18} />
              <span>Settings</span>
            </NavLink>
          </li>
        </ul>
        <div className="sidebar-user">
          <div className="sidebar-avatar">{initial}</div>
          <span className="sidebar-email">{user.email}</span>
        </div>
        <button type="button" className="btn btn-secondary btn-sm" onClick={handleLogout} style={{ width: '100%' }}>
          Logout
        </button>
      </div>
    )
  }

  return (
    <div className="layout">
      <button
        type="button"
        className="sidebar-toggle"
        onClick={() => setSidebarOpen(o => !o)}
        aria-label={sidebarOpen ? 'Close sidebar' : 'Open sidebar'}
        aria-expanded={sidebarOpen}
        aria-controls="sidebar-nav"
      >
        {sidebarOpen ? '✕' : '☰'}
      </button>
      <nav id="sidebar-nav" className={`sidebar${sidebarOpen ? ' sidebar-open' : ''}`}>
        <div className="sidebar-logo">
          <div className="sidebar-logo-icon">B</div>
          <span>BigBase</span>
        </div>

        <NavSection title="Overview" items={[{ to: '/', label: 'Dashboard', icon: 'layout-dashboard', end: true }]} />
        <NavSection title="Build" items={buildNav} />
        <NavSection title="Data" items={dataNav} />
        <NavSection title="Auth" items={authNav} />
        <NavSection title="Engage" items={engageNav} />
        <NavSection title="DevOps" items={devOpsNav} />

        <div className="sidebar-spacer" />
        {footerBlock}
      </nav>

      <div className="layout-body">
        <main className="content">
          <Outlet />
        </main>

        <footer className="app-footer" data-testid="app-footer">
          <div className="app-footer-brand">
            <span className="app-footer-logo" aria-hidden>B</span>
            <span>© 2026 BigBase · MIT License</span>
          </div>
          <div className="app-footer-meta">
            {appVersion && <span className="mono">v{appVersion}</span>}
            <span className="app-footer-sep" aria-hidden />
            <span>
              Built with{' '}
              <a href="https://github.com/danielvm-git/bigpowers" target="_blank" rel="noopener noreferrer">
                BigPowers
              </a>
              {' '}by{' '}
              <a href="https://github.com/danielvm-git" target="_blank" rel="noopener noreferrer">
                danielvm-git
              </a>
            </span>
            <span className="app-footer-sep" aria-hidden />
            <a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noopener noreferrer">
              GitHub
            </a>
            <a
              href="https://github.com/danielvm-git/bigbase/blob/main/CHANGELOG.md"
              target="_blank"
              rel="noopener noreferrer"
            >
              Changelog
            </a>
          </div>
        </footer>
      </div>
    </div>
  )
}
