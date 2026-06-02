import type { ReactNode } from 'react'

interface StreamLogProps {
  logs: string[]
  isStreaming?: boolean
  className?: string
}

export function StreamLog({ logs, isStreaming = false, className = '' }: StreamLogProps): ReactNode {
  if (logs.length === 0 && !isStreaming) {
    return <p className="dim" data-testid="stream-log-empty">No logs</p>
  }

  return (
    <div className={`stream-log ${className}`} data-testid="stream-log">
      {logs.map((line, i) => (
        <div key={i} className="stream-log-line" data-testid="stream-log-line">
          <span className="stream-log-ln">{i + 1}</span>
          <code className="stream-log-text">{line}</code>
        </div>
      ))}
      {isStreaming && (
        <div className="stream-log-line stream-log-cursor" data-testid="stream-log-cursor">
          <span className="stream-log-ln">&nbsp;</span>
          <code className="stream-log-text">▊</code>
        </div>
      )}
    </div>
  )
}
