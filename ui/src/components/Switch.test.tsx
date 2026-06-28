import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { Switch } from './Switch'

describe('Switch', () => {
  it('renders with role=switch', () => {
    render(<Switch label="Enable notifications" />)
    expect(screen.getByRole('switch')).toBeInTheDocument()
  })

  it('has aria-checked=false by default', () => {
    render(<Switch label="Enable" />)
    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'false')
  })

  it('has aria-checked=true when checked', () => {
    render(<Switch label="Enable" checked={true} onChange={vi.fn()} />)
    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'true')
  })

  it('toggles on click', () => {
    const onChange = vi.fn()
    render(<Switch label="Toggle me" onChange={onChange} />)
    fireEvent.click(screen.getByRole('switch'))
    expect(onChange).toHaveBeenCalledOnce()
  })

  it('toggles on Space key', () => {
    const onChange = vi.fn()
    render(<Switch label="Toggle me" onChange={onChange} />)
    fireEvent.keyDown(screen.getByRole('switch'), { key: ' ' })
    expect(onChange).toHaveBeenCalledOnce()
  })

  it('applies sm size class', () => {
    render(<Switch label="Small" size="sm" />)
    expect(screen.getByRole('switch').className).toContain('switch-sm')
  })

  it('applies md size class by default', () => {
    render(<Switch label="Default" />)
    expect(screen.getByRole('switch').className).toContain('switch-md')
  })

  it('renders disabled state', () => {
    render(<Switch label="Disabled" disabled />)
    expect(screen.getByRole('switch')).toHaveAttribute('aria-disabled', 'true')
  })

  it('associates with label text', () => {
    render(<Switch label="Dark mode" />)
    expect(screen.getByText('Dark mode')).toBeInTheDocument()
  })
})
