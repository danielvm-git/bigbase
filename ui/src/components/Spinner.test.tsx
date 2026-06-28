import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Spinner } from './Spinner'

describe('Spinner', () => {
  it('has role=status', () => {
    render(<Spinner />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('has default aria-label', () => {
    render(<Spinner />)
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Loading')
  })

  it('accepts custom aria-label', () => {
    render(<Spinner aria-label="Saving..." />)
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Saving...')
  })

  it('applies sm size class', () => {
    render(<Spinner size="sm" />)
    expect(screen.getByRole('status').className).toContain('spinner-sm')
  })

  it('applies md size class by default', () => {
    render(<Spinner />)
    expect(screen.getByRole('status').className).toContain('spinner-md')
  })

  it('applies lg size class', () => {
    render(<Spinner size="lg" />)
    expect(screen.getByRole('status').className).toContain('spinner-lg')
  })
})
