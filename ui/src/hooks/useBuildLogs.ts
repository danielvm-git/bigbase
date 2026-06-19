import { useState, useEffect, useCallback } from 'react'

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

  useEffect(() => {
    fetchLogs()
  }, [fetchLogs])

  useEffect(() => {
    if (!deploymentId || (status !== 'pending' && status !== 'building')) {
      return
    }

    const interval = setInterval(() => {
      fetchLogs(true)
    }, 2000)

    return () => clearInterval(interval)
  }, [deploymentId, status, fetchLogs])

  return { lines, loading, error, status, refresh: fetchLogs }
}
