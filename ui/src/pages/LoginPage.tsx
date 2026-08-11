import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Input, Card } from '../components'

interface AuthResponse {
  token: string
  error?: string
}

function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
}

export default function LoginPage() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [emailError, setEmailError] = useState('')
  const [passwordError, setPasswordError] = useState('')
  const [isRegister, setIsRegister] = useState(false)
  const [googleEnabled, setGoogleEnabled] = useState(false)
  const [showForgot, setShowForgot] = useState(false)
  const [resetEmail, setResetEmail] = useState('')
  const [resetEmailError, setResetEmailError] = useState('')
  const [resetSent, setResetSent] = useState(false)

  useEffect(() => {
    fetch('/api/auth/oauth/google', { redirect: 'manual' })
      .then(r => { if (r.status === 302) setGoogleEnabled(true) })
      .catch(() => {})
  }, [])

  const validateLogin = (): boolean => {
    let ok = true
    if (!email.trim()) {
      setEmailError('Email is required')
      ok = false
    } else if (!isValidEmail(email)) {
      setEmailError('Enter a valid email address')
      ok = false
    } else {
      setEmailError('')
    }
    if (!password) {
      setPasswordError('Password is required')
      ok = false
    } else if (password.length < 6) {
      setPasswordError('Password must be at least 6 characters')
      ok = false
    } else {
      setPasswordError('')
    }
    return ok
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!validateLogin()) return

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

  const handleForgotSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!resetEmail.trim()) {
      setResetEmailError('Email is required')
      return
    }
    if (!isValidEmail(resetEmail)) {
      setResetEmailError('Enter a valid email address')
      return
    }
    setResetEmailError('')
    setResetSent(true)
  }

  if (showForgot) {
    return (
      <div className="login-page">
        <Card className="login-card">
          <div className="login-brand">
            <h1>Reset password</h1>
            <p>Enter your email and we will send reset instructions.</p>
          </div>
          {resetSent ? (
            <p className="dim">If an account exists for {resetEmail}, you will receive an email shortly.</p>
          ) : (
            <form onSubmit={handleForgotSubmit} className="login-form">
              <Input
                type="email"
                label="Email"
                placeholder="Email"
                value={resetEmail}
                onChange={e => { setResetEmail(e.target.value); setResetEmailError('') }}
                error={resetEmailError}
              />
              <Button type="submit" variant="primary">Send reset link</Button>
            </form>
          )}
          <p className="login-toggle">
            <button className="btn-link" type="button" onClick={() => { setShowForgot(false); setResetSent(false) }}>
              Back to sign in
            </button>
          </p>
        </Card>
      </div>
    )
  }

  return (
    <div className="login-page">
      <Card className="login-card">
        <div className="login-brand">
          <div className="login-brand-logo">B</div>
          <h1>BigBase</h1>
          <p>{isRegister ? 'Create your account' : 'Sign in to continue'}</p>
        </div>

        <form onSubmit={handleSubmit} className="login-form">
          <Input
            type="email"
            label="Email"
            placeholder="Email"
            value={email}
            onChange={e => { setEmail(e.target.value); setEmailError('') }}
            error={emailError}
          />
          <Input
            type="password"
            label="Password"
            placeholder="Password"
            value={password}
            onChange={e => { setPassword(e.target.value); setPasswordError('') }}
            error={passwordError}
          />
          {!isRegister && (
            <p style={{ textAlign: 'right', marginTop: '-var(--space-2)' }}>
              <button type="button" className="btn-link" onClick={() => setShowForgot(true)}>
                Forgot password?
              </button>
            </p>
          )}
          {error && <p className="input-error-text">{error}</p>}
          <Button type="submit" variant="primary">
            {isRegister ? 'Register' : 'Sign In'}
          </Button>
        </form>

        {googleEnabled && (
          <>
            <div className="divider"><span>or</span></div>
            <a href="/api/auth/oauth/google" className="google-btn">
              <svg viewBox="0 0 24 24" width="18" height="18"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>
              Sign in with Google
            </a>
          </>
        )}

        <p className="login-toggle">
          {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
          <button className="btn-link" onClick={() => setIsRegister(!isRegister)} type="button">
            {isRegister ? 'Sign In' : 'Register'}
          </button>
        </p>
      </Card>
    </div>
  )
}
