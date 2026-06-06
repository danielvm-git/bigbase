export interface SystemMetricsSlice {
  cpu_percent: number
  memory_mb: number
  goroutines: number
  uptime_seconds: number
}

export interface HostMetricsSlice {
  cpu_percent: number
  mem_used_bytes: number
  mem_total_bytes: number
  disk_used_bytes: number
  disk_total_bytes: number
  net_bytes_in: number
  net_bytes_out: number
  collected_at: string
}

export interface MonitoringMetricsPayload {
  system?: SystemMetricsSlice
  host?: HostMetricsSlice
  requests?: {
    total: number
    avg_latency_ms: number
    by_status: Record<string, number>
    by_endpoint: Record<string, { count: number; status_count: Record<string, number>; avg_latency_ms: number }>
  }
}

export interface SSEMetricsEvent {
  system: SystemMetricsSlice
  host: HostMetricsSlice
  sent_at: string
}

/** Fetch /api/monitoring/metrics (CPU needs a prior server sample for non-zero percent). */
export async function fetchMonitoringMetrics(signal?: AbortSignal): Promise<MonitoringMetricsPayload> {
  const res = await fetch('/api/monitoring/metrics', { signal })
  if (!res.ok) return {}
  return res.json() as Promise<MonitoringMetricsPayload>
}

/**
 * Second fetch after a short gap so the backend can compute CPU delta (first sample is always 0).
 */
export async function fetchMonitoringMetricsWarmed(
  signal?: AbortSignal,
  gapMs = 200,
): Promise<MonitoringMetricsPayload> {
  const first = await fetchMonitoringMetrics(signal)
  if (signal?.aborted) return first
  await new Promise(resolve => setTimeout(resolve, gapMs))
  if (signal?.aborted) return first
  return fetchMonitoringMetrics(signal)
}
