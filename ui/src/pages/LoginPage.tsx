import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

interface AuthResponse {
  token: string
  error?: string
}

export default function LoginPage() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isRegister, setIsRegister] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    const endpoint = isRegister ? '/api/auth/register' : '/api/auth/login'
    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
      const data = (await res.json()) as AuthResponse
      if (!res.ok) {
        setError(data.error || 'request failed')
        return
      }
      nav('/')
    } catch {
      setError('network error')
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>BigBase</h1>
        <h2>{isRegister ? 'Create Account' : 'Sign In'}</h2>
        <form onSubmit={handleSubmit}>
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            required
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            required
            minLength={6}
          />
          {error && <p className="error">{error}</p>}
          <button type="submit">{isRegister ? 'Register' : 'Sign In'}</button>
        </form>
        <p className="toggle">
          {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
          <button className="link" onClick={() => setIsRegister(!isRegister)}>
            {isRegister ? 'Sign In' : 'Register'}
          </button>
        </p>
      </div>
    </div>
  )
}
