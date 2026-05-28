import { useEffect, useState } from 'react'
import { PageHeader, Button } from '../components'

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
    fetchUsers()
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

  if (loading) return <div className="loading">Loading users...</div>

  return (
    <div>
      <PageHeader title="Users">
        <Button variant="secondary" size="sm" onClick={fetchUsers}>Refresh</Button>
      </PageHeader>
      {error && <p className="input-error-text">{error}</p>}
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
                    <Button variant="danger" size="sm" onClick={() => handleDelete(u.id)}>Delete</Button>
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
