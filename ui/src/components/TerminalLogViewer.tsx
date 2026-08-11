import { type ReactNode } from 'react'
import { useBuildLogs } from '../hooks/useBuildLogs'
import { StreamLog } from './StreamLog'

interface TerminalLogViewerProps {
  deploymentId: string
}

export function TerminalLogViewer({ deploymentId }: TerminalLogViewerProps): ReactNode {
  const { lines, loading, error, isStreaming } = useBuildLogs(deploymentId)

  if (loading && lines.length === 0) {
    return (
      <div className="stream-log-container" data-testid="terminal-log-viewer-loading" role="status" aria-busy="true">
        <div className="stream-log-toolbar">
          <span className="dim" style={{ padding: 'var(--space-2)' }}>Connecting...</span>
        </div>
        <div className="stream-log" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <p className="dim">Loading logs...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="card-error" style={{ padding: 'var(--space-4)' }} data-testid="terminal-log-viewer-error" role="alert">
        <p className="input-error-text">{error}</p>
      </div>
    )
  }

  return (
    <div data-testid="terminal-log-viewer">
      {isStreaming && (
        <div className="stream-log-badge" data-testid="terminal-log-viewer-live" role="status">
          ● LIVE
        </div>
      )}
      <StreamLog
        logs={lines}
        isStreaming={isStreaming}
        showToolbar={true}
      />
    </div>
  )
}
