import { useRef, useEffect } from 'react'

interface BuildLogsProps {
  lines: string[]
  loading?: boolean
  error?: string | null
}

export function BuildLogs({ lines, loading, error }: BuildLogsProps) {
  const preRef = useRef<HTMLPreElement>(null)

  useEffect(() => {
    if (preRef.current) {
      preRef.current.scrollTop = preRef.current.scrollHeight
    }
  }, [lines])

  if (loading && lines.length === 0) {
    return <p className="dim" role="status" aria-busy="true">Loading logs...</p>
  }

  if (error) {
    return (
      <div className="card-error" role="alert" style={{ padding: 'var(--space-4)' }}>
        <p className="input-error-text">{error}</p>
      </div>
    )
  }

  if (lines.length === 0) {
    return <p className="dim">No build logs available.</p>
  }

  return (
    <pre
      ref={preRef}
      role="log"
      aria-live="polite"
      aria-label="Build log"
      tabIndex={0}
      style={{
        background: 'var(--bg-code, #1e1e1e)',
        color: '#d4d4d4',
        padding: 'var(--space-4)',
        borderRadius: 'var(--radius-s)',
        fontSize: 'var(--text-xs)',
        fontFamily: 'var(--font-mono)',
        overflowY: 'auto',
        maxHeight: '400px',
        lineHeight: '1.5',
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-all'
      }}
    >
      {lines.map((line, i) => (
        <div key={i}>{line}</div>
      ))}
    </pre>
  )
}
