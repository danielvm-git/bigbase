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
})

describe('statusBadgeVariant', () => {
  it.each([
    ['running', 'success'],
    ['active', 'success'],
    ['healthy', 'success'],
    ['failed', 'error'],
    ['error', 'error'],
    ['building', 'warning'],
    ['pending', 'warning'],
    ['unknown', 'neutral'],
  ])('maps status "%s" to variant "%s"', (status, expected) => {
    expect(statusBadgeVariant(status)).toBe(expected)
  })
})
