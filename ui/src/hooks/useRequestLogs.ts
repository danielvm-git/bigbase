import { useState, useEffect, useCallback } from 'react'

export interface RequestLogEntry {
  id: string
  method: string
  path: string
  status: number
  duration_ms: number
  created_at: string
}

export function useRequestLogs(siteId: string) {
  const [logs, setLogs] = useState<RequestLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  const [pathPrefix, setPathPrefix] = useState('')
  const [statusClass, setStatusClass] = useState('') // '', 2xx, 4xx, 5xx
  const [limit, setLimit] = useState(100)

  const fetchLogs = useCallback(async () => {
    if (!siteId) return

    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams()
      if (pathPrefix) params.set('path', pathPrefix)
      if (statusClass) params.set('status_class', statusClass)
      if (limit) params.set('limit', limit.toString())

      const res = await fetch(`/api/sites/${siteId}/logs?${params.toString()}`)
      const data = await res.json()

      if (!res.ok) {
        setError(data.error || `Failed to fetch request logs (HTTP ${res.status})`)
        return
      }

      setLogs(data.data || [])
      setTotal(data.total || 0)
    } catch (err) {
      setError('Failed to fetch request logs — network error')
    } finally {
      setLoading(false)
    }
  }, [siteId, pathPrefix, statusClass, limit])

  useEffect(() => {
    fetchLogs()
  }, [fetchLogs])

  return {
    logs,
    total,
    loading,
    error,
    pathPrefix,
    setPathPrefix,
    statusClass,
    setStatusClass,
    limit,
    setLimit,
    refresh: fetchLogs
  }
}
