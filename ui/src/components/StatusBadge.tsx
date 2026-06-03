import type { StatusKind } from './Badge'

const DEFAULT_LABELS: Record<StatusKind, string> = {
  ready: 'Ready',
  building: 'Building',
  failed: 'Failed',
  pending: 'Pending',
}

export interface StatusBadgeProps {
  status: StatusKind
  label?: string
}

/**
 * Status indicator with a word+dot/spinner — never color-alone.
 * Required by prototype design system (a11y constraint: status must be
 * communicated by both color AND text/shape).
 */
export function StatusBadge({ status, label }: StatusBadgeProps) {
  const text = label ?? DEFAULT_LABELS[status]
  const isBuilding = status === 'building'

  return (
    <span
      role="status"
      aria-label={text}
      className={`status-badge status-badge-${status}`}
    >
      <span
        aria-hidden="true"
        className={`status-indicator-${status} ${isBuilding ? 'status-spinner' : 'status-dot'}`}
      />
      <span className="status-badge-label">{text}</span>
    </span>
  )
}
