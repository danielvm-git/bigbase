import type { ReactNode } from 'react'

interface SparklineProps {
  values: number[]
  width?: number
  height?: number
  color?: string
  label?: string
}

function summarize(values: number[]): string {
  if (values.length === 0) return 'sparkline: no data'
  const min = Math.min(...values)
  const max = Math.max(...values)
  const last = values[values.length - 1]
  return `sparkline: ${values.length} data points, min ${min}, max ${max}, last ${last}`
}

/**
 * A minimal SVG sparkline. Renders the last `values.length` data points
 * as a polyline against a transparent background.
 */
export function Sparkline({ values, width = 120, height = 40, color = '#3b82f6', label }: SparklineProps): ReactNode {
  if (values.length < 2) {
    return (
      <svg width={width} height={height} role="img" aria-label={label ?? summarize(values)}>
        <line x1={0} y1={height / 2} x2={width} y2={height / 2} stroke={color} strokeWidth={1} opacity={0.3} />
      </svg>
    )
  }

  const max = Math.max(...values, 1)
  const step = width / (values.length - 1)

  const points = values
    .map((v, i) => {
      const x = i * step
      const y = height - (v / max) * (height - 4) - 2
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')

  return (
    <svg width={width} height={height} role="img" aria-label={label ?? summarize(values)} style={{ overflow: 'visible' }}>
      <polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  )
}
