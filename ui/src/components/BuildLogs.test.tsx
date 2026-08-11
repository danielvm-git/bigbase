import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BuildLogs } from './BuildLogs'

describe('BuildLogs', () => {
  it('renders log lines', () => {
    render(<BuildLogs lines={['line one', 'line two']} />)

    expect(screen.getByText('line one')).toBeInTheDocument()
    expect(screen.getByText('line two')).toBeInTheDocument()
  })

  it('exposes the log region with live-region semantics and keyboard scroll', () => {
    render(<BuildLogs lines={['building...']} />)

    const log = screen.getByRole('log', { name: 'Build log' })
    expect(log).toHaveAttribute('aria-live', 'polite')
    expect(log).toHaveAttribute('tabindex', '0')
  })

  it('announces loading state as a busy status message', () => {
    render(<BuildLogs lines={[]} loading={true} />)

    expect(screen.getByRole('status', { busy: true })).toHaveTextContent('Loading logs...')
  })

  it('announces errors as an alert', () => {
    render(<BuildLogs lines={[]} error="Build failed" />)

    expect(screen.getByRole('alert')).toHaveTextContent('Build failed')
  })

  it('shows empty state when there are no lines', () => {
    render(<BuildLogs lines={[]} />)

    expect(screen.getByText('No build logs available.')).toBeInTheDocument()
  })
})
