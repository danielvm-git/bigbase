import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { CopyButton } from './CopyButton'

describe('CopyButton', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'clipboard', {
      writable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
  })

  it('renders copy button', () => {
    render(<CopyButton value="some text" />)
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('has accessible label', () => {
    render(<CopyButton value="some text" />)
    expect(screen.getByRole('button')).toHaveAttribute('aria-label', expect.stringContaining('Copy'))
  })

  it('calls clipboard.writeText on click', async () => {
    render(<CopyButton value="copy me" />)
    fireEvent.click(screen.getByRole('button'))
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('copy me')
  })

  it('shows copied feedback after click', async () => {
    render(<CopyButton value="copy me" />)
    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => screen.getByText(/copied/i))
    expect(screen.getByText(/copied/i)).toBeInTheDocument()
  })

  it('renders custom label', () => {
    render(<CopyButton value="x" label="Copy token" />)
    expect(screen.getByText('Copy token')).toBeInTheDocument()
  })
})
