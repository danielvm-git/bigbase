import { useEffect, useRef, useState, useCallback, type ReactNode } from 'react'
import { ansiToHtml } from '../lib/ansi'
import { Button } from './Button'

interface StreamLogProps {
  logs: string[]
  isStreaming?: boolean
  className?: string
  autoScroll?: boolean
  showToolbar?: boolean
  showTimestamps?: boolean
  searchQuery?: string
  onSearchChange?: (q: string) => void
}

function formatTimestamp(): string {
  const now = new Date()
  return now.toTimeString().slice(0, 8)
}

export function StreamLog({
  logs,
  isStreaming = false,
  className = '',
  autoScroll = true,
  showToolbar = true,
  showTimestamps = false,
  searchQuery: externalSearchQuery,
  onSearchChange,
}: StreamLogProps): ReactNode {
  const scrollRef = useRef<HTMLDivElement>(null)
  const [internalSearch, setInternalSearch] = useState('')
  const [timestamps, setTimestamps] = useState(false)
  const [copied, setCopied] = useState(false)

  const searchQuery = externalSearchQuery ?? internalSearch
  const handleSearchChange = onSearchChange ?? setInternalSearch

  const filteredLogs = searchQuery
    ? logs.filter(line => line.toLowerCase().includes(searchQuery.toLowerCase()))
    : logs

  useEffect(() => {
    if (!autoScroll || !scrollRef.current) return
    scrollRef.current.scrollTop = scrollRef.current.scrollHeight
  }, [filteredLogs, isStreaming, autoScroll])

  const handleCopy = useCallback(async () => {
    const text = logs.join('\n')
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [logs])

  const showTimestampsEffective = showTimestamps || timestamps

  if (logs.length === 0 && !isStreaming) {
    return <p className="dim" data-testid="stream-log-empty">No logs</p>
  }

  return (
    <div className={`stream-log-container ${className}`} data-testid="stream-log-container">
      {showToolbar && (
        <div className="stream-log-toolbar" data-testid="stream-log-toolbar">
          <input
            type="text"
            className="stream-log-search"
            placeholder="Filter logs..."
            value={searchQuery}
            onChange={e => handleSearchChange(e.target.value)}
            data-testid="stream-log-search"
          />
          <div className="stream-log-toolbar-actions">
            <button
              className={`stream-log-toolbar-btn ${showTimestampsEffective ? 'stream-log-toolbar-btn--active' : ''}`}
              onClick={() => setTimestamps(t => !t)}
              title="Toggle timestamps"
              data-testid="stream-log-timestamp-btn"
            >
              🕐
            </button>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCopy}
              title="Copy all logs"
              data-testid="stream-log-copy-btn"
            >
              {copied ? '✓ Copied' : '📋 Copy'}
            </Button>
          </div>
        </div>
      )}
      <div ref={scrollRef} className="stream-log" data-testid="stream-log">
        {filteredLogs.map((line, i) => {
          const html = ansiToHtml(line)
          return (
            <div key={`line-${i}-${line.slice(0, 12)}`} className="stream-log-line" data-testid="stream-log-line">
              <span className="stream-log-ln">{i + 1}</span>
              {showTimestampsEffective && (
                <span className="stream-log-ts">{formatTimestamp()}</span>
              )}
              <code
                className="stream-log-text"
                dangerouslySetInnerHTML={{ __html: html || '&nbsp;' }}
              />
            </div>
          )
        })}
        {isStreaming && (
          <div className="stream-log-line stream-log-cursor" data-testid="stream-log-cursor">
            <span className="stream-log-ln">&nbsp;</span>
            {showTimestampsEffective && (
              <span className="stream-log-ts">&nbsp;</span>
            )}
            <code className="stream-log-text">▊</code>
          </div>
        )}
      </div>
    </div>
  )
}
