import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MetricCard } from './MetricCard'

describe('MetricCard', () => {
  it('renders label, value, and subtitle', () => {
    render(<MetricCard label="CPU" value="23%" subtitle="goroutines" />)
    expect(screen.getByText('CPU')).toBeInTheDocument()
    expect(screen.getByText('23%')).toBeInTheDocument()
    expect(screen.getByText('goroutines')).toBeInTheDocument()
  })

  it('applies success color', () => {
    render(<MetricCard label="Ok" value={42} color="success" />)
    const valueEl = screen.getByText('42')
    expect(valueEl.style.color).toBe('var(--success-fg)')
  })

  it('applies error color', () => {
    render(<MetricCard label="Errors" value={5} color="error" />)
    const valueEl = screen.getByText('5')
    expect(valueEl.style.color).toBe('var(--error-fg)')
  })

  it('shows up trend arrow', () => {
    render(<MetricCard label="Growth" value="15%" trend="up" />)
    expect(screen.getByText('↑')).toBeInTheDocument()
  })

  it('shows secondary value when provided', () => {
    render(<MetricCard label="Memory" value="512 MB" secondaryValue="1.2 GB total" />)
    expect(screen.getByText('1.2 GB total')).toBeInTheDocument()
  })

  it('handles numeric zero correctly', () => {
    render(<MetricCard label="Errors" value={0} color="error" />)
    expect(screen.getByText('0')).toBeInTheDocument()
  })
})
