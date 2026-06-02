import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ComponentHealthGrid } from './ComponentHealthGrid'

describe('ComponentHealthGrid', () => {
  it('shows empty state when no components', () => {
    render(<ComponentHealthGrid components={[]} />)
    expect(screen.getByText('No component data available')).toBeInTheDocument()
  })

  it('renders components with correct status badges', () => {
    const comps = [
      { name: 'auth', status: 'healthy' as const },
      { name: 'db', status: 'degraded' as const },
      { name: 'cache', status: 'down' as const },
      { name: 'queue', status: 'unknown' as const },
    ]
    render(<ComponentHealthGrid components={comps} />)

    expect(screen.getByText('auth')).toBeInTheDocument()
    expect(screen.getByText('db')).toBeInTheDocument()
    expect(screen.getByText('cache')).toBeInTheDocument()
    expect(screen.getByText('queue')).toBeInTheDocument()

    expect(screen.getByText('healthy')).toBeInTheDocument()
    expect(screen.getByText('degraded')).toBeInTheDocument()
    expect(screen.getByText('down')).toBeInTheDocument()
    expect(screen.getByText('unknown')).toBeInTheDocument()
  })

  it('displays version when provided', () => {
    const comps = [{ name: 'auth', status: 'healthy' as const, version: '1.2.3' }]
    render(<ComponentHealthGrid components={comps} />)

    expect(screen.getByText('v1.2.3')).toBeInTheDocument()
  })

  it('does not display version when absent', () => {
    const comps = [{ name: 'db', status: 'healthy' as const }]
    render(<ComponentHealthGrid components={comps} />)

    expect(screen.queryByText(/^v/)).not.toBeInTheDocument()
  })
})
