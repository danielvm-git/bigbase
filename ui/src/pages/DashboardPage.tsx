import { useEffect, useState } from 'react'

interface JwtPayload {
  user_id: number
  email: string
}

export default function DashboardPage() {
  const [user, setUser] = useState<{ id: number; email: string } | null>(null)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) return
    try {
      const payload = JSON.parse(atob(token.split('.')[1])) as JwtPayload
      setUser({ id: payload.user_id, email: payload.email })
    } catch {
      localStorage.removeItem('token')
    }
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
