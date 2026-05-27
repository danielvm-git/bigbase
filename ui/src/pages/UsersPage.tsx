import { useEffect, useState } from 'react'

interface User {
  id: number
  email: string
  created_at: string
}

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchUsers = async () => {
    setError('')
    setLoading(true)
    try {
      const res = await fetch('/api/auth/users')
      const data = await res.json()
      if (!res.ok) {
        setError(data.error || `error: ${res.status}`)
        setUsers([])
        return
      }
      setUsers((data as { data: User[] }).data || [])
    } catch {
      setError('network error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const controller = new AbortController()
    fetchUsers()
    return () => controller.abort()
  }, [])

  const handleDelete = async (id: number) => {
    if (!confirm(`Delete user #${id}?`)) return
    try {
      const res = await fetch(`/api/auth/users/${id}`, { method: 'DELETE' })
      if (!res.ok) {
        const data = await res.json()
        setError(data.error || `error: ${res.status}`)
        return
      }
      setUsers(prev => prev.filter(u => u.id !== id))
    } catch {
      setError('network error')
    }
  }

  if (loading) return <p className="loading">Loading users...</p>

  return (
    <div className="users-page">
      <div className="users-header">
        <h1>Users</h1>
        <button className="refresh-btn" onClick={fetchUsers}>Refresh</button>
      </div>
      {error && <p className="error">{error}</p>}
      {users.length === 0 && !error && <p className="dim">No users found.</p>}
      {users.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Email</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id}>
                  <td>{u.id}</td>
                  <td>{u.email}</td>
                  <td>{new Date(u.created_at).toLocaleString()}</td>
                  <td>
                    <button className="delete-btn" onClick={() => handleDelete(u.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
