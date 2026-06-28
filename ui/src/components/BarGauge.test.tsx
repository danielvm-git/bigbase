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
})
