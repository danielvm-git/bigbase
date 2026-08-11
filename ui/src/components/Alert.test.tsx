import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { Alert } from './Alert'

describe('Alert', () => {
  it('renders message', () => {
    render(<Alert>Something happened</Alert>)
    expect(screen.getByText('Something happened')).toBeInTheDocument()
  })

  it('has role=alert', () => {
    render(<Alert>Info</Alert>)
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })

  it('applies info variant by default', () => {
    render(<Alert>Info</Alert>)
    expect(screen.getByRole('alert').className).toContain('alert-info')
  })

  it('applies success variant', () => {
    render(<Alert variant="success">Done</Alert>)
    expect(screen.getByRole('alert').className).toContain('alert-success')
  })

  it('applies warning variant', () => {
    render(<Alert variant="warning">Warning</Alert>)
    expect(screen.getByRole('alert').className).toContain('alert-warning')
  })

  it('applies error variant', () => {
    render(<Alert variant="error">Error</Alert>)
    expect(screen.getByRole('alert').className).toContain('alert-error')
  })

  it('renders title when provided', () => {
    render(<Alert title="Heads up">Message</Alert>)
    expect(screen.getByText('Heads up')).toBeInTheDocument()
  })

  it('renders dismiss button when dismissible', () => {
    render(<Alert dismissible onDismiss={vi.fn()}>Alert</Alert>)
    expect(screen.getByRole('button', { name: /dismiss/i })).toBeInTheDocument()
  })

  it('calls onDismiss when dismiss button clicked', () => {
    const onDismiss = vi.fn()
    render(<Alert dismissible onDismiss={onDismiss}>Alert</Alert>)
    fireEvent.click(screen.getByRole('button', { name: /dismiss/i }))
    expect(onDismiss).toHaveBeenCalledOnce()
  })

  it('does not render dismiss button when not dismissible', () => {
    render(<Alert>Alert</Alert>)
    expect(screen.queryByRole('button', { name: /dismiss/i })).not.toBeInTheDocument()
  })

  it.each([
    ['info', 'Info:'],
    ['success', 'Success:'],
    ['warning', 'Warning:'],
    ['error', 'Error:'],
  ] as const)('renders a visible %s status word (non-color cue)', (variant, word) => {
    render(<Alert variant={variant}>Message</Alert>)
    const label = screen.getByText(word)
    expect(label).toBeInTheDocument()
    expect(label).not.toHaveAttribute('aria-hidden')
  })
})
