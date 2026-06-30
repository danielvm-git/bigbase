import { useEffect, useState } from 'react'
import { Outlet, useNavigate } from 'react-router-dom'
import { Icon, type IconName } from './components/Icon'
import { ThemePicker } from './components/ThemePicker'
import { TutorialOverlay, useTutorialTrigger } from './components/TutorialOverlay'
import { AppShell } from './components/AppShell'
import { Sidebar, SidebarSection } from './components/Sidebar'
import { AppFooter } from './components/AppFooter'
import { useTheme } from './hooks/useTheme'

interface UserInfo { id: number; email: string }

interface NavItem {
  to: string
  label: string
  icon: IconName
  end?: boolean
}

export default function Layout() {
  const nav = useNavigate()
  const { theme, toggle: toggleTheme, accent, setAccent } = useTheme()
  const [user, setUser] = useState<UserInfo | null>(null)
  const [appVersion, setAppVersion] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const tutorial = useTutorialTrigger()

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
    { to: '/events', label: 'Events', icon: 'zap' },
  ]

  const sidebarFooter = user ? (
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
          <a href="/settings">
            <Icon name="settings" size={18} />
            <span>Settings</span>
          </a>
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
  ) : null

  const sidebar = (
    <Sidebar id="sidebar-nav" open={sidebarOpen} footer={sidebarFooter}>
      <div className="sidebar-logo">
        <div className="sidebar-logo-icon">B</div>
        <span>BigBase</span>
      </div>
      <SidebarSection title="Overview" items={[{ to: '/', label: 'Dashboard', icon: 'layout-dashboard', end: true }]} />
      <SidebarSection title="Build" items={buildNav} />
      <SidebarSection title="Data" items={dataNav} />
      <SidebarSection title="Auth" items={authNav} />
      <SidebarSection title="Engage" items={engageNav} />
      <SidebarSection title="DevOps" items={devOpsNav} />
    </Sidebar>
  )

  return (
    <AppShell
      sidebar={sidebar}
      sidebarOpen={sidebarOpen}
      onToggleSidebar={() => setSidebarOpen(o => !o)}
    >
      {tutorial.visible && <TutorialOverlay onClose={tutorial.close} />}
      <main className="content">
        <Outlet />
      </main>
      <AppFooter
        appVersion={appVersion}
        showTutorial={!tutorial.isDone}
        onOpenTutorial={tutorial.open}
      />
    </AppShell>
  )
}
