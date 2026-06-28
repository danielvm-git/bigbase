import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Label } from './Label'

describe('Label', () => {
  it('renders label text', () => {
    render(<Label htmlFor="email">Email</Label>)
    expect(screen.getByText('Email')).toBeInTheDocument()
  })

  it('sets htmlFor attribute', () => {
    render(<Label htmlFor="email">Email</Label>)
    expect(screen.getByText('Email').closest('label')).toHaveAttribute('for', 'email')
  })

  it('shows required indicator when required prop is set', () => {
    render(<Label htmlFor="name" required>Name</Label>)
    expect(screen.getByText('*')).toBeInTheDocument()
  })

  it('does not show required indicator by default', () => {
    render(<Label htmlFor="name">Name</Label>)
    expect(screen.queryByText('*')).not.toBeInTheDocument()
  })

  it('applies custom className', () => {
    render(<Label htmlFor="x" className="custom-label">Label</Label>)
    expect(screen.getByText('Label').closest('label')?.className).toContain('custom-label')
  })
})
