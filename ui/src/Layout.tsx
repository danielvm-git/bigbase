import { useEffect } from 'react'
import { Outlet, Link, useNavigate } from 'react-router-dom'

export default function Layout() {
  const nav = useNavigate()

  useEffect(() => {
    if (!localStorage.getItem('token')) {
      nav('/login')
    }
  }, [nav])

  const handleLogout = () => {
    localStorage.removeItem('token')
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
