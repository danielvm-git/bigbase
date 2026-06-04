/** Relative time label for deployment timestamps. */
export function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.floor(hrs / 24)
  if (days < 30) return `${days}d ago`
  return new Date(iso).toLocaleDateString()
}

const THUMB_COLORS = [
  'linear-gradient(135deg, #6366f1 0%, #4f46e5 100%)',
  'linear-gradient(135deg, #374151 0%, #1f2937 100%)',
  'linear-gradient(135deg, #dc2626 0%, #991b1b 100%)',
  'linear-gradient(135deg, #059669 0%, #047857 100%)',
  'linear-gradient(135deg, #7c3aed 0%, #5b21b6 100%)',
]

export function siteThumbStyle(siteId: string): { background: string } {
  let h = 0
  for (let i = 0; i < siteId.length; i++) h = (h + siteId.charCodeAt(i)) % THUMB_COLORS.length
  return { background: THUMB_COLORS[h] }
}

export function siteDisplayUrl(name: string): string {
  const slug = name.replace(/[^a-z0-9-]/gi, '-').toLowerCase() || 'site'
  return `${slug}.bigbase.local`
}

export function mapDeployStatus(status: string): string {
  if (status === 'running') return 'ready'
  if (status === 'pending' || status === 'building') return 'building'
  if (status === 'failed') return 'failed'
  return status
}
