import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { AreaChart } from './AreaChart'

const series = [
  {
    label: 'Requests',
    color: '#4F46E5',
    data: [
      { x: '2026-01-01', y: 100 },
      { x: '2026-01-02', y: 150 },
      { x: '2026-01-03', y: 120 },
    ],
  },
]

describe('AreaChart', () => {
  it('renders an SVG', () => {
    const { container } = render(<AreaChart series={series} />)
    expect(container.querySelector('svg')).toBeInTheDocument()
  })

  it('renders accessible data table', () => {
    render(<AreaChart series={series} />)
    expect(screen.getByRole('table')).toBeInTheDocument()
  })

  it('renders series label in legend', () => {
    const { container } = render(<AreaChart series={series} />)
    const legend = container.querySelector('.area-chart-legend')
    expect(legend?.textContent).toContain('Requests')
  })

  it('renders data values in table', () => {
    render(<AreaChart series={series} />)
    expect(screen.getByText('100')).toBeInTheDocument()
    expect(screen.getByText('150')).toBeInTheDocument()
  })

  it('renders x-axis labels in data table', () => {
    const { container } = render(<AreaChart series={series} />)
    const tds = container.querySelectorAll('td')
    const texts = Array.from(tds).map(td => td.textContent)
    expect(texts).toContain('2026-01-01')
  })

  it('renders title when provided', () => {
    const { container } = render(<AreaChart series={series} title="Traffic" />)
    expect(container.querySelector('.area-chart-title')?.textContent).toBe('Traffic')
  })

  it('renders empty message when no data', () => {
    render(<AreaChart series={[]} emptyMessage="No data" />)
    expect(screen.getByText('No data')).toBeInTheDocument()
  })

  it('renders multiple series', () => {
    const multi = [
      { label: 'Read', color: '#10B981', data: [{ x: 'A', y: 10 }] },
      { label: 'Write', color: '#F59E0B', data: [{ x: 'A', y: 5 }] },
    ]
    const { container } = render(<AreaChart series={multi} />)
    const legend = container.querySelector('.area-chart-legend')
    expect(legend?.textContent).toContain('Read')
    expect(legend?.textContent).toContain('Write')
  })
})
