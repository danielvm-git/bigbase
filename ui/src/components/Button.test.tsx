import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Button } from './Button'

describe('Button', () => {
  it('renders with default variant and size', () => {
    render(<Button>Click me</Button>)
    const btn = screen.getByRole('button', { name: 'Click me' })
    expect(btn.className).toContain('btn')
    expect(btn.className).toContain('btn-primary')
  })

  it('applies secondary variant', () => {
    render(<Button variant="secondary">Secondary</Button>)
    expect(screen.getByRole('button').className).toContain('btn-secondary')
  })

  it('applies danger variant', () => {
    render(<Button variant="danger">Delete</Button>)
    expect(screen.getByRole('button').className).toContain('btn-danger')
  })

  it('applies ghost variant', () => {
    render(<Button variant="ghost">Cancel</Button>)
    expect(screen.getByRole('button').className).toContain('btn-ghost')
  })

  it('applies link variant', () => {
    render(<Button variant="link">Link</Button>)
    expect(screen.getByRole('button').className).toContain('btn-link')
  })

  it('applies sm size', () => {
    render(<Button size="sm">Small</Button>)
    expect(screen.getByRole('button').className).toContain('btn-sm')
  })

  it('applies block size for full-width', () => {
    render(<Button size="block">Wide</Button>)
    const btn = screen.getByRole('button', { name: 'Wide' })
    expect(btn.className).toContain('btn-block')
  })

  it('renders as anchor when as="a"', () => {
    render(<Button as="a" href="/x">Anchor</Button>)
    const a = screen.getByRole('link', { name: 'Anchor' })
    expect(a).toBeInTheDocument()
  })
})
