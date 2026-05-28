import { useEffect, useState } from 'react'

interface Message {
  id: string
  channel: string
  to_addr: string
  subject: string
  body: string
  status: string
  created_at: string
}

type Channel = 'email' | 'sms' | 'push'

export default function MessagingPage() {
  const [tab, setTab] = useState<Channel>('email')
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)

  // Email
  const [emailTo, setEmailTo] = useState('')
  const [emailSubject, setEmailSubject] = useState('')
  const [emailBody, setEmailBody] = useState('')
  // SMS
  const [smsTo, setSmsTo] = useState('')
  const [smsMsg, setSmsMsg] = useState('')
  // Push
  const [pushToken, setPushToken] = useState('')
  const [pushTitle, setPushTitle] = useState('')
  const [pushBody, setPushBody] = useState('')

  const fetchMessages = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/messaging/messages')
      const d = await res.json()
      if (!res.ok) { setError(d.error || `error: ${res.status}`); setMessages([]) }
      else { setMessages((d as { data: Message[] }).data || []) }
    } catch { setError('network error') }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchMessages() }, [])

  const sendEmail = async (e: React.FormEvent) => {
    e.preventDefault(); setSending(true); setError('')
    try {
      const res = await fetch('/api/messaging/email', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ to: emailTo, subject: emailSubject, body: emailBody }),
      })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'send failed'); setSending(false); return }
      setEmailTo(''); setEmailSubject(''); setEmailBody('')
      fetchMessages()
    } catch { setError('network error') }
    finally { setSending(false) }
  }

  const sendSms = async (e: React.FormEvent) => {
    e.preventDefault(); setSending(true); setError('')
    try {
      const res = await fetch('/api/messaging/sms', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ to: smsTo, message: smsMsg }),
      })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'send failed'); setSending(false); return }
      setSmsTo(''); setSmsMsg('')
      fetchMessages()
    } catch { setError('network error') }
    finally { setSending(false) }
  }

  const sendPush = async (e: React.FormEvent) => {
    e.preventDefault(); setSending(true); setError('')
    try {
      const res = await fetch('/api/messaging/push', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: pushToken, title: pushTitle, body: pushBody }),
      })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'send failed'); setSending(false); return }
      setPushToken(''); setPushTitle(''); setPushBody('')
      fetchMessages()
    } catch { setError('network error') }
    finally { setSending(false) }
  }

  const tabs: Channel[] = ['email', 'sms', 'push']

  return (
    <div className="page">
      <div className="page-header">
        <h1>Messaging</h1>
        <button className="refresh-btn" onClick={fetchMessages}>Refresh</button>
      </div>
      {error && <p className="error">{error}</p>}

      <div className="tabs">
        {tabs.map(t => (
          <button key={t} className={`tab ${tab === t ? 'active' : ''}`} onClick={() => setTab(t)}>
            {t.toUpperCase()}
          </button>
        ))}
      </div>

      <div className="card">
        {tab === 'email' && (
          <form onSubmit={sendEmail} className="msg-form">
            <input placeholder="To" value={emailTo} onChange={e => setEmailTo(e.target.value)} required />
            <input placeholder="Subject" value={emailSubject} onChange={e => setEmailSubject(e.target.value)} />
            <textarea placeholder="Body" value={emailBody} onChange={e => setEmailBody(e.target.value)} required rows={4} />
            <button type="submit" className="create-btn" disabled={sending}>{sending ? 'Sending...' : 'Send Email'}</button>
          </form>
        )}
        {tab === 'sms' && (
          <form onSubmit={sendSms} className="msg-form">
            <input placeholder="To" value={smsTo} onChange={e => setSmsTo(e.target.value)} required />
            <textarea placeholder="Message" value={smsMsg} onChange={e => setSmsMsg(e.target.value)} required rows={3} />
            <button type="submit" className="create-btn" disabled={sending}>{sending ? 'Sending...' : 'Send SMS'}</button>
          </form>
        )}
        {tab === 'push' && (
          <form onSubmit={sendPush} className="msg-form">
            <input placeholder="Device Token" value={pushToken} onChange={e => setPushToken(e.target.value)} required />
            <input placeholder="Title" value={pushTitle} onChange={e => setPushTitle(e.target.value)} />
            <textarea placeholder="Body" value={pushBody} onChange={e => setPushBody(e.target.value)} required rows={3} />
            <button type="submit" className="create-btn" disabled={sending}>{sending ? 'Sending...' : 'Send Push'}</button>
          </form>
        )}
      </div>

      <h2 className="section-title">History</h2>
      {loading ? <p className="loading">Loading...</p> : messages.length === 0 ? <p className="dim">No messages sent.</p> : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Channel</th>
                <th>To</th>
                <th>Subject</th>
                <th>Status</th>
                <th>Sent</th>
              </tr>
            </thead>
            <tbody>
              {messages.map(m => (
                <tr key={m.id}>
                  <td><span className={`channel-badge channel-${m.channel}`}>{m.channel}</span></td>
                  <td>{m.to_addr}</td>
                  <td className="dim">{m.subject || '—'}</td>
                  <td>{m.status}</td>
                  <td>{new Date(m.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
