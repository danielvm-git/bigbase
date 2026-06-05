import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MetricCard } from './MetricCard'

describe('MetricCard', () => {
  it('renders label and value', () => {
    render(<MetricCard label="CPU" value="42%" />)
    expect(screen.getByText('CPU')).toBeInTheDocument()
    expect(screen.getByText('42%')).toBeInTheDocument()
  })

  it('renders optional subtitle', () => {
    render(<MetricCard label="CPU" value="42%" subtitle="across 4 cores" />)
    expect(screen.getByText('across 4 cores')).toBeInTheDocument()
  })

  it('renders trend arrow when trend is provided', () => {
    const { rerender } = render(<MetricCard label="CPU" value="42%" trend="up" />)
    expect(screen.getByText('↑')).toBeInTheDocument()

    rerender(<MetricCard label="CPU" value="42%" trend="down" />)
    expect(screen.getByText('↓')).toBeInTheDocument()

    rerender(<MetricCard label="CPU" value="42%" trend="flat" />)
    expect(screen.getByText('→')).toBeInTheDocument()
  })

  it('renders secondary value when provided', () => {
    render(<MetricCard label="Memory" value="512 MB" secondaryValue="25% used" />)
    expect(screen.getByText('25% used')).toBeInTheDocument()
  })
})
