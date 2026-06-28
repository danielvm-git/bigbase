interface DataPoint {
  x: string
  y: number
}

interface Series {
  label: string
  color: string
  data: DataPoint[]
}

interface AreaChartProps {
  series: Series[]
  title?: string
  width?: number
  height?: number
  emptyMessage?: string
  className?: string
}

export function AreaChart({
  series,
  title,
  width = 600,
  height = 200,
  emptyMessage = 'No data available.',
  className = '',
}: AreaChartProps) {
  const padL = 40, padR = 16, padT = 16, padB = 32
  const chartW = width - padL - padR
  const chartH = height - padT - padB

  if (series.length === 0 || series.every(s => s.data.length === 0)) {
    return (
      <div className={`area-chart area-chart-empty ${className}`.trim()}>
        {title && <p className="area-chart-title">{title}</p>}
        <p className="area-chart-empty-msg">{emptyMessage}</p>
      </div>
    )
  }

  const allPoints = series.flatMap(s => s.data)
  const allY = allPoints.map(p => p.y)
  const minY = Math.min(0, ...allY)
  const maxY = Math.max(...allY) || 1
  const allX = [...new Set(allPoints.map(p => p.x))].sort()

  function xPos(x: string): number {
    const idx = allX.indexOf(x)
    return padL + (idx / Math.max(allX.length - 1, 1)) * chartW
  }

  function yPos(y: number): number {
    return padT + chartH - ((y - minY) / (maxY - minY)) * chartH
  }

  function buildPath(data: DataPoint[]): string {
    if (data.length === 0) return ''
    const pts = data.map(p => `${xPos(p.x)},${yPos(p.y)}`)
    const lastX = xPos(data[data.length - 1].x)
    const firstX = xPos(data[0].x)
    const baseY = padT + chartH
    return `M${firstX},${baseY} L${pts.join(' L')} L${lastX},${baseY} Z`
  }

  function buildLine(data: DataPoint[]): string {
    if (data.length === 0) return ''
    return data.map((p, i) => `${i === 0 ? 'M' : 'L'}${xPos(p.x)},${yPos(p.y)}`).join(' ')
  }

  const tickCount = Math.min(allX.length, 5)
  const tickStep = Math.floor(allX.length / (tickCount - 1 || 1))
  const xTicks = allX.filter((_, i) => i % (tickStep || 1) === 0)

  return (
    <div className={`area-chart ${className}`.trim()}>
      {title && <p className="area-chart-title">{title}</p>}

      <div className="area-chart-legend" aria-hidden="true">
        {series.map(s => (
          <span key={s.label} className="area-chart-legend-item" aria-hidden="true">
            <span className="area-chart-legend-dot" style={{ background: s.color }} />
            {s.label}
          </span>
        ))}
      </div>

      <svg
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        aria-hidden="true"
        className="area-chart-svg"
      >
        {series.map(s => (
          <g key={s.label}>
            <path d={buildPath(s.data)} fill={s.color} fillOpacity={0.15} stroke="none" />
            <path d={buildLine(s.data)} fill="none" stroke={s.color} strokeWidth={2} />
          </g>
        ))}
        {xTicks.map(x => (
          <text
            key={x}
            x={xPos(x)}
            y={padT + chartH + 20}
            textAnchor="middle"
            fontSize={11}
            fill="var(--fg-tertiary, #9999)"
          >
            {x}
          </text>
        ))}
      </svg>

      <table
        className="area-chart-data-table"
        aria-label={title ?? 'Chart data'}
        style={{ position: 'absolute', left: '-9999px' }}
      >
        <thead>
          <tr>
            <th>Date</th>
            {series.map(s => <th key={s.label}>{s.label}</th>)}
          </tr>
        </thead>
        <tbody>
          {allX.map(x => (
            <tr key={x}>
              <td>{x}</td>
              {series.map(s => {
                const pt = s.data.find(p => p.x === x)
                return <td key={s.label}>{pt?.y ?? '—'}</td>
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
