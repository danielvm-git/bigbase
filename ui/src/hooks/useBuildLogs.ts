import { useState, useEffect, useCallback, useRef } from 'react'

export interface BuildLogResponse {
  deployment_id: string
  status: string
  lines: string[]
  log_available: boolean
  error_message?: string
}

export function useBuildLogs(deploymentId: string) {
  const [lines, setLines] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)

  const fetchLogs = useCallback(async (isPolling = false) => {
    if (!deploymentId) return

    if (!isPolling) setLoading(true)
    setError(null)

    try {
      const res = await fetch(`/api/deploy/${deploymentId}/logs`)
      const data = await res.json()

      if (!res.ok) {
        setError(data.error || `Failed to fetch logs (HTTP ${res.status})`)
        return
      }

      setLines(data.lines || [])
      setStatus(data.status)
    } catch (err) {
      if (!isPolling) setError('Failed to fetch logs — network error')
    } finally {
      if (!isPolling) setLoading(false)
    }
  }, [deploymentId])

  // Initial fetch + WebSocket streaming
  useEffect(() => {
    if (!deploymentId) return

    let ws: WebSocket | null = null
    let closed = false
    setIsStreaming(false)

    // Always do an initial fetch for baseline data
    fetchLogs()

    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      ws = new WebSocket(`${protocol}//${window.location.host}/api/deploy/${deploymentId}/logs/stream`)
      wsRef.current = ws

      ws.onopen = () => {
        if (closed) return
        setIsStreaming(true)
        setLoading(false)
      }

      ws.onmessage = (event: MessageEvent) => {
        if (closed) return
        const line = typeof event.data === 'string' ? event.data : ''
        if (line) {
          setLines(prev => [...prev, line])
        }
      }

      ws.onclose = () => {
        if (closed) return
        setIsStreaming(false)
        wsRef.current = null
        // Clean close — done streaming
      }

      ws.onerror = () => {
        if (closed) return
        setIsStreaming(false)
        wsRef.current = null
        // Close the failed socket silently
        ws?.close()
      }
    } catch {
      // WebSocket not available — already fetched via initial fetch above
    }

    return () => {
      closed = true
      if (ws && ws.readyState === 1) { // OPEN
        ws.close()
      }
      wsRef.current = null
    }
  }, [deploymentId, fetchLogs])

  // Polling (when not streaming via WebSocket)
  useEffect(() => {
    if (!deploymentId || isStreaming || (status !== 'pending' && status !== 'building' && status !== 'deploying')) {
      return
    }

    const interval = setInterval(() => {
      fetchLogs(true)
    }, 2000)

    return () => clearInterval(interval)
  }, [deploymentId, status, isStreaming, fetchLogs])

  return { lines, loading, error, status, isStreaming, refresh: fetchLogs }
}
