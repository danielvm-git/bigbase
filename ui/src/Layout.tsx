import { useEffect } from 'react'
import { Outlet, Link, useNavigate } from 'react-router-dom'

export default function Layout() {
  const nav = useNavigate()

  useEffect(() => {
    fetch('/api/auth/me')
      .then(r => { if (!r.ok) nav('/login') })
      .catch(() => nav('/login'))
  }, [nav])

  const handleLogout = () => {
    nav('/login')
  }

  return (
    <div className="layout">
      <nav className="sidebar">
        <h2 className="logo">BigBase</h2>
        <ul>
          <li><Link to="/">Dashboard</Link></li>
          <li><Link to="/data">Data Studio</Link></li>
          <li><Link to="/sql">SQL Editor</Link></li>
          <li><Link to="/users">Users</Link></li>
          <li><Link to="/repos">Git Repos</Link></li>
          <li><Link to="/deploy">Deploy</Link></li>
          <li><Link to="/messaging">Messaging</Link></li>
          <li><Link to="/storage">Storage</Link></li>
          <li><Link to="/functions">Functions</Link></li>
          <li><Link to="/forge">Forge</Link></li>
          <li><Link to="/cici">CI/CD</Link></li>
          <li><Link to="/monitoring">Monitoring</Link></li>
        </ul>
        <div className="spacer" />
        <button className="logout-btn" onClick={handleLogout}>Logout</button>
      </nav>
      <main className="content">
        <Outlet />
      </main>
    </div>
  )
}
