import type { ReactNode } from 'react'

interface DonutGaugeProps {
  used: number
  total: number
  label?: string
  color?: string
  size?: number
}

/**
 * A minimal SVG donut gauge showing used/total as a circular arc.
 * Values are in arbitrary units; only the ratio matters.
 */
export function DonutGauge({ used, total, label, color = '#3b82f6', size = 80 }: DonutGaugeProps): ReactNode {
  const pct = total > 0 ? Math.min(used / total, 1) : 0
  const r = (size - 10) / 2
  const cx = size / 2
  const cy = size / 2
  const circumference = 2 * Math.PI * r

  // Arc length for the used portion.
  const dash = pct * circumference
  const gap = circumference - dash

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      role="img"
      aria-label={label ?? 'gauge'}
      style={{ display: 'block' }}
    >
      {/* Background track */}
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke="var(--color-border, #e5e7eb)"
        strokeWidth={8}
      />
      {/* Used arc — rotated so it starts at the top */}
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke={color}
        strokeWidth={8}
        strokeDasharray={`${dash} ${gap}`}
        strokeLinecap="round"
        transform={`rotate(-90 ${cx} ${cy})`}
        style={{ transition: 'stroke-dasharray 0.4s ease' }}
      />
      {/* Center label */}
      <text
        x={cx}
        y={cy + 5}
        textAnchor="middle"
        fontSize="12"
        fill="currentColor"
        fontWeight="600"
      >
        {(pct * 100).toFixed(0)}%
      </text>
    </svg>
  )
}
