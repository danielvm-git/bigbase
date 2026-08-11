import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { BarGauge } from './BarGauge'

describe('BarGauge — existing API', () => {
  it('renders label', () => {
    render(<BarGauge used={50} total={100} label="Memory" />)
    expect(screen.getByText('Memory')).toBeInTheDocument()
  })

  it('renders used/total value', () => {
    render(<BarGauge used={50} total={100} label="Disk" />)
    expect(screen.getByText('50 / 100')).toBeInTheDocument()
  })

  it('renders without label', () => {
    const { container } = render(<BarGauge used={25} total={100} />)
    expect(container.firstChild).toBeInTheDocument()
  })

  it('uses formatValue when provided', () => {
    render(<BarGauge used={512} total={1024} label="Storage" formatValue={n => `${n}MB`} />)
    expect(screen.getByText('512MB / 1024MB')).toBeInTheDocument()
  })

  it('renders single bar as progressbar with data-bearing name and values', () => {
    render(<BarGauge used={50} total={100} label="Memory" />)
    const bar = screen.getByRole('progressbar', { name: 'Memory: 50 of 100' })
    expect(bar).toHaveAttribute('aria-valuenow', '50')
    expect(bar).toHaveAttribute('aria-valuemin', '0')
    expect(bar).toHaveAttribute('aria-valuemax', '100')
  })

  it('names the progressbar with the data even without a label', () => {
    render(<BarGauge used={25} total={100} />)
    expect(screen.getByRole('progressbar', { name: '25 of 100' })).toBeInTheDocument()
  })

  it('clamps aria-valuenow to total', () => {
    render(<BarGauge used={150} total={100} label="Disk" />)
    const bar = screen.getByRole('progressbar')
    expect(bar).toHaveAttribute('aria-valuenow', '100')
    expect(bar).toHaveAttribute('aria-valuemax', '100')
  })
})

describe('BarGauge — segments mode', () => {
  const segments = [
    { value: 30, label: 'Reads', color: '#10B981' },
    { value: 20, label: 'Writes', color: '#F59E0B' },
  ]

  it('renders stacked segment labels', () => {
    render(<BarGauge used={0} total={100} mode="stacked" segments={segments} />)
    expect(screen.getByText('Reads')).toBeInTheDocument()
    expect(screen.getByText('Writes')).toBeInTheDocument()
  })

  it('renders grouped segment labels', () => {
    render(<BarGauge used={0} total={100} mode="grouped" segments={segments} />)
    expect(screen.getByText('Reads')).toBeInTheDocument()
    expect(screen.getByText('Writes')).toBeInTheDocument()
  })

  it('renders segment values', () => {
    render(<BarGauge used={0} total={100} mode="stacked" segments={segments} />)
    expect(screen.getByText('30')).toBeInTheDocument()
    expect(screen.getByText('20')).toBeInTheDocument()
  })

  it('gives stacked segments data-bearing accessible names', () => {
    render(<BarGauge used={0} total={100} mode="stacked" segments={segments} />)
    expect(screen.getByRole('img', { name: 'Reads: 30' })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Writes: 20' })).toBeInTheDocument()
  })

  it('gives grouped segments data-bearing accessible names', () => {
    render(<BarGauge used={0} total={100} mode="grouped" segments={segments} />)
    expect(screen.getByRole('img', { name: 'Reads: 30' })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Writes: 20' })).toBeInTheDocument()
  })
})
