import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  it('renders auto-derived label for ready status', () => {
    render(<StatusBadge status="ready" />)
    expect(screen.getByText('Ready')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Ready')
  })

  it('renders auto-derived label for building status', () => {
    render(<StatusBadge status="building" />)
    expect(screen.getByText('Building')).toBeInTheDocument()
  })

  it('renders auto-derived label for failed status', () => {
    render(<StatusBadge status="failed" />)
    expect(screen.getByText('Failed')).toBeInTheDocument()
  })

  it('renders auto-derived label for pending status', () => {
    render(<StatusBadge status="pending" />)
    expect(screen.getByText('Pending')).toBeInTheDocument()
  })

  it('overrides label when provided', () => {
    render(<StatusBadge status="ready" label="Deployed" />)
    expect(screen.getByText('Deployed')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Deployed')
  })

  it('renders a dot for non-building states and a spinner for building', () => {
    const { rerender, container } = render(<StatusBadge status="ready" />)
    expect(container.querySelector('.status-dot')).toBeInTheDocument()
    expect(container.querySelector('.status-spinner')).not.toBeInTheDocument()

    rerender(<StatusBadge status="building" />)
    expect(container.querySelector('.status-spinner')).toBeInTheDocument()
    expect(container.querySelector('.status-dot')).not.toBeInTheDocument()
  })

  it('applies the status-specific color class to the indicator', () => {
    const { container, rerender } = render(<StatusBadge status="ready" />)
    expect(container.querySelector('.status-indicator-ready')).toBeInTheDocument()

    rerender(<StatusBadge status="building" />)
    expect(container.querySelector('.status-indicator-building')).toBeInTheDocument()

    rerender(<StatusBadge status="failed" />)
    expect(container.querySelector('.status-indicator-failed')).toBeInTheDocument()

    rerender(<StatusBadge status="pending" />)
    expect(container.querySelector('.status-indicator-pending')).toBeInTheDocument()
  })

  it('does not rely on color alone (text label is always rendered)', () => {
    // a11y invariant: every status has a text label, not just a colored dot
    const { rerender } = render(<StatusBadge status="ready" />)
    expect(screen.getByText('Ready')).toBeInTheDocument()
    rerender(<StatusBadge status="failed" />)
    expect(screen.getByText('Failed')).toBeInTheDocument()
  })
})
