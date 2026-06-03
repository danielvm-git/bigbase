import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Modal } from './Modal'

describe('Modal', () => {
  it('renders title and body when open', () => {
    render(
      <Modal open title="Confirm" onClose={() => {}}>
        <p>Body text</p>
      </Modal>,
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Confirm')).toBeInTheDocument()
    expect(screen.getByText('Body text')).toBeInTheDocument()
  })

  it('does not render when closed', () => {
    render(
      <Modal open={false} title="Hidden" onClose={() => {}}>
        <p>Nope</p>
      </Modal>,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('calls onClose when Escape is pressed', () => {
    const onClose = vi.fn()
    render(
      <Modal open title="Dialog" onClose={onClose}>
        <button type="button">Inside</button>
      </Modal>,
    )
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('returns focus to trigger after close', () => {
    const onClose = vi.fn()
    const { rerender } = render(
      <>
        <button type="button">Trigger</button>
        <Modal open={false} title="Dialog" onClose={onClose}>
          <p>Content</p>
        </Modal>
      </>,
    )
    const trigger = screen.getByRole('button', { name: 'Trigger' })
    trigger.focus()
    rerender(
      <>
        <button type="button">Trigger</button>
        <Modal open title="Dialog" onClose={onClose}>
          <p>Content</p>
        </Modal>
      </>,
    )
    rerender(
      <>
        <button type="button">Trigger</button>
        <Modal open={false} title="Dialog" onClose={onClose}>
          <p>Content</p>
        </Modal>
      </>,
    )
    expect(document.activeElement).toBe(trigger)
  })
})
