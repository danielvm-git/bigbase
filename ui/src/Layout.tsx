import { useEffect, useState } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'

interface UserInfo { id: number; email: string }

function SidebarIcon({ children }: { children: string }) {
  return <span className="sidebar-nav-icon">{children}</span>
}

export default function Layout() {
  const nav = useNavigate()
  const [user, setUser] = useState<UserInfo | null>(null)
  const [appVersion, setAppVersion] = useState<string | null>(null)

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

  return (
    <div className="layout">
      <nav className="sidebar">
        <div className="sidebar-logo">
          <div className="sidebar-logo-icon">B</div>
          <span>BigBase</span>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-section-title">Overview</div>
          <ul className="sidebar-nav">
            <li>
              <NavLink to="/" end>
                <SidebarIcon>H</SidebarIcon>
                <span>Dashboard</span>
              </NavLink>
            </li>
          </ul>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-section-title">Data</div>
          <ul className="sidebar-nav">
            <li>
              <NavLink to="/data">
                <SidebarIcon>D</SidebarIcon>
                <span>Data Studio</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/sql">
                <SidebarIcon>S</SidebarIcon>
                <span>SQL Editor</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/storage">
                <SidebarIcon>F</SidebarIcon>
                <span>Storage</span>
              </NavLink>
            </li>
          </ul>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-section-title">Services</div>
          <ul className="sidebar-nav">
            <li>
              <NavLink to="/users">
                <SidebarIcon>U</SidebarIcon>
                <span>Users</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/repos">
                <SidebarIcon>G</SidebarIcon>
                <span>Git Repos</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/deploy">
                <SidebarIcon>R</SidebarIcon>
                <span>Deploy</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/functions">
                <SidebarIcon>λ</SidebarIcon>
                <span>Functions</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/messaging">
                <SidebarIcon>M</SidebarIcon>
                <span>Messaging</span>
              </NavLink>
            </li>
          </ul>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-section-title">DevOps</div>
          <ul className="sidebar-nav">
            <li>
              <NavLink to="/forge">
                <SidebarIcon>I</SidebarIcon>
                <span>Forge</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/cici">
                <SidebarIcon>C</SidebarIcon>
                <span>CI/CD</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/monitoring">
                <SidebarIcon>V</SidebarIcon>
                <span>Monitoring</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/realtime">
                <SidebarIcon>R</SidebarIcon>
                <span>Realtime</span>
              </NavLink>
            </li>
          </ul>
        </div>

        <div className="sidebar-spacer" />

        {user && (
          <div className="sidebar-footer">
            <div className="sidebar-user">
              <div className="sidebar-avatar">{initial}</div>
              <span className="sidebar-email">{user.email}</span>
            </div>
            <button className="btn btn-secondary btn-sm" onClick={handleLogout} style={{ width: '100%' }}>
              Logout
            </button>
          </div>
        )}
      </nav>

      <div className="layout-body">
        <main className="content">
          <Outlet />
        </main>

        <footer className="page-footer">
          {appVersion && <span>BigBase v{appVersion}</span>}
        </footer>
      </div>
    </div>
  )
}
