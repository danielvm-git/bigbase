import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components'

interface BusEvent {
  hook: string
  data: Record<string, unknown>
  timestamp: string
}

export default function EventsPage() {
  const [events, setEvents] = useState<BusEvent[]>([])
  const [connected, setConnected] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const es = new EventSource('/api/monitoring/events')

    es.onopen = () => setConnected(true)
    es.onerror = () => setConnected(false)

    es.onmessage = (e) => {
      try {
        const parsed = JSON.parse(e.data) as BusEvent
        setEvents(prev => {
          const next = [parsed, ...prev]
          // Keep a rolling window of 200 events
          return next.length > 200 ? next.slice(0, 200) : next
        })
      } catch {
        // ignore parse errors
      }
    }

    return () => {
      es.close()
      setConnected(false)
    }
  }, [])

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = 0
    }
  }, [events.length])

  return (
    <div className="events-page">
      <PageHeader
        title="Event Bus"
        subtitle="Live stream of internal event bus emissions."
      >
        <span role="status" className={`badge badge-${connected ? 'success' : 'warning'}`}>
          {connected ? 'Connected' : 'Connecting…'}
        </span>
      </PageHeader>

      <div ref={listRef} className="events-log" data-testid="events-log" aria-live="polite">
        {events.length === 0 ? (
          <p className="dim events-empty">Waiting for events…</p>
        ) : (
          events.map((ev, i) => (
            <div key={i} className="events-row">
              <span className="events-ts mono dim">{ev.timestamp}</span>
              <span className="events-hook badge badge-info">{ev.hook}</span>
              <span className="events-data mono dim">
                {JSON.stringify(ev.data)}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
