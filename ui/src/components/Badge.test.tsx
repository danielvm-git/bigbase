import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Badge, statusBadgeVariant } from './Badge'

describe('Badge', () => {
  it('renders with correct text', () => {
    render(<Badge>Active</Badge>)
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('applies default neutral variant class', () => {
    render(<Badge>Neutral</Badge>)
    const el = screen.getByText('Neutral')
    expect(el.className).toContain('badge-neutral')
  })

  it('applies success variant class when specified', () => {
    render(<Badge variant="success">OK</Badge>)
    const el = screen.getByText('OK')
    expect(el.className).toContain('badge-success')
  })

  it('applies warning variant', () => {
    render(<Badge variant="warning">Warn</Badge>)
    expect(screen.getByText('Warn').className).toContain('badge-warning')
  })

  it('applies error variant', () => {
    render(<Badge variant="error">Fail</Badge>)
    expect(screen.getByText('Fail').className).toContain('badge-error')
  })

  it('applies accent variant', () => {
    render(<Badge variant="accent">New</Badge>)
    expect(screen.getByText('New').className).toContain('badge-accent')
  })
})

describe('statusBadgeVariant', () => {
  it.each([
    ['running', 'success'],
    ['active', 'success'],
    ['ok', 'success'],
    ['healthy', 'success'],
    ['failed', 'error'],
    ['error', 'error'],
    ['deleted', 'error'],
    ['building', 'warning'],
    ['pending', 'warning'],
    ['deploying', 'warning'],
    ['unknown', 'neutral'],
  ])('maps status "%s" to variant "%s"', (status, expected) => {
    expect(statusBadgeVariant(status)).toBe(expected)
  })
})
