import { useEffect, useState } from 'react'

export default function DashboardPage() {
  const [user, setUser] = useState<{ id: number; email: string } | null>(null)

  useEffect(() => {
    fetch('/api/auth/me')
      .then(r => r.ok ? r.json() : null)
      .then(d => setUser(d as { id: number; email: string } | null))
      .catch(() => setUser(null))
  }, [])

  if (!user) {
    return <p className="loading">Loading...</p>
  }

  return (
    <div className="dashboard">
      <h1>Dashboard</h1>
      <div className="card">
        <h3>Signed in as</h3>
        <p className="email">{user.email}</p>
      </div>
      <div className="card">
        <h3>User ID</h3>
        <p className="id">#{user.id}</p>
      </div>
    </div>
  )
}
