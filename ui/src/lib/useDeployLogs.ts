import { useCallback, useEffect, useRef, useState } from 'react'

export interface DeployLogsResponse {
  deployment_id: string
  status: string
  lines: string[]
  log_available?: boolean
  error_message?: string
}

export interface UseDeployLogsOptions {
  deploymentId: string
  enabled: boolean
  pollIntervalMs?: number
  previewLines?: string[]
  /** Status from the main deploy poll; preferred for streaming cursor when set. */
  deployStatus?: string
}

export interface UseDeployLogsResult {
  lines: string[]
  status: string
  errorMessage: string
  fetchError: string
  isStreaming: boolean
  loading: boolean
}

export async function fetchDeployLogs(deploymentId: string): Promise<DeployLogsResponse | null> {
  try {
    const res = await fetch(`/api/deploy/${deploymentId}/logs`)
    if (!res.ok) return null
    const data = (await res.json()) as DeployLogsResponse & { lines?: string[] }
    return {
      deployment_id: data.deployment_id ?? deploymentId,
      status: data.status ?? '',
      lines: Array.isArray(data.lines) ? data.lines : [],
      log_available: data.log_available,
      error_message: data.error_message,
    }
  } catch {
    return null
  }
}

function isTerminalDeployStatus(status: string): boolean {
  return status === 'running' || status === 'failed'
}

export function useDeployLogs({
  deploymentId,
  enabled,
  pollIntervalMs = 1500,
  previewLines,
  deployStatus = '',
}: UseDeployLogsOptions): UseDeployLogsResult {
  const [lines, setLines] = useState<string[]>(previewLines ?? [])
  const [status, setStatus] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [fetchError, setFetchError] = useState('')
  const [loading, setLoading] = useState(false)
  const previewRef = useRef(previewLines)

  previewRef.current = previewLines

  const refresh = useCallback(async () => {
    if (!deploymentId) return
    setLoading(true)
    try {
      const data = await fetchDeployLogs(deploymentId)
      if (!data) {
        setFetchError('Could not load build logs')
        return
      }
      setFetchError('')
      setLines(data.lines)
      if (data.status) setStatus(data.status)
      if (data.error_message) setErrorMessage(data.error_message)
    } finally {
      setLoading(false)
    }
  }, [deploymentId])

  useEffect(() => {
    if (!deploymentId) {
      setLines(previewRef.current ?? [])
      setFetchError('')
      return
    }
    if (!enabled) {
      void refresh()
      return
    }
    void refresh()
    const id = setInterval(() => void refresh(), pollIntervalMs)
    return () => clearInterval(id)
  }, [deploymentId, enabled, pollIntervalMs, refresh])

  useEffect(() => {
    if (previewLines && !deploymentId) {
      setLines(previewLines)
    }
  }, [previewLines, deploymentId])

  const activeStatus = deployStatus || status
  const isStreaming =
    enabled &&
    deploymentId !== '' &&
    !isTerminalDeployStatus(activeStatus)

  return { lines, status, errorMessage, fetchError, isStreaming, loading }
}
