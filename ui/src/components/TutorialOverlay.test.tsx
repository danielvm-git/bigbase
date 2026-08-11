import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TutorialOverlay } from './TutorialOverlay'

describe('TutorialOverlay', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('renders the dialog with title, body, and described-by link', () => {
    render(<TutorialOverlay onClose={() => {}} />)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(screen.getByText('Welcome to BigBase')).toBeInTheDocument()
    const body = document.getElementById('tutorial-description')
    expect(body).toBeInTheDocument()
    expect(body).toHaveTextContent(/BigBase is a self-hosted/)
    expect(dialog).toHaveAttribute('aria-describedby', 'tutorial-description')
  })

  it('moves focus into the dialog on open', () => {
    render(<TutorialOverlay onClose={() => {}} />)
    expect(document.activeElement).toBe(
      screen.getByRole('button', { name: 'Close tutorial' }),
    )
  })

  it('calls onClose when Escape is pressed', () => {
    const onClose = vi.fn()
    render(<TutorialOverlay onClose={onClose} />)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('wraps Tab from last focusable back to first', () => {
    render(<TutorialOverlay onClose={() => {}} />)
    const first = screen.getByRole('button', { name: 'Close tutorial' })
    const last = screen.getByRole('button', { name: 'Next' })
    last.focus()
    fireEvent.keyDown(document, { key: 'Tab' })
    expect(document.activeElement).toBe(first)
  })

  it('wraps Shift+Tab from first focusable to last', () => {
    render(<TutorialOverlay onClose={() => {}} />)
    const first = screen.getByRole('button', { name: 'Close tutorial' })
    const last = screen.getByRole('button', { name: 'Next' })
    first.focus()
    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(last)
  })

  it('returns focus to the trigger after close', () => {
    const onClose = vi.fn()
    const { rerender } = render(<button type="button">Trigger</button>)
    const trigger = screen.getByRole('button', { name: 'Trigger' })
    trigger.focus()
    rerender(
      <>
        <button type="button">Trigger</button>
        <TutorialOverlay onClose={onClose} />
      </>,
    )
    rerender(<button type="button">Trigger</button>)
    expect(document.activeElement).toBe(trigger)
  })
})
