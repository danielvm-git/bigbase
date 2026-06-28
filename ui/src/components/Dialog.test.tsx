import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { Dialog } from './Dialog'

describe('Dialog', () => {
  it('does not render when closed', () => {
    render(<Dialog open={false} title="Confirm" onClose={vi.fn()}><p>Content</p></Dialog>)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('renders when open', () => {
    render(<Dialog open={true} title="Confirm" onClose={vi.fn()}><p>Content</p></Dialog>)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('renders title', () => {
    render(<Dialog open={true} title="Are you sure?" onClose={vi.fn()}><p>x</p></Dialog>)
    expect(screen.getByText('Are you sure?')).toBeInTheDocument()
  })

  it('renders children', () => {
    render(<Dialog open={true} title="Confirm" onClose={vi.fn()}><p>Dialog body</p></Dialog>)
    expect(screen.getByText('Dialog body')).toBeInTheDocument()
  })

  it('calls onClose when close button clicked', () => {
    const onClose = vi.fn()
    render(<Dialog open={true} title="Confirm" onClose={onClose}><p>x</p></Dialog>)
    fireEvent.click(screen.getByRole('button', { name: /close/i }))
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('calls onClose on Escape key', () => {
    const onClose = vi.fn()
    render(<Dialog open={true} title="Confirm" onClose={onClose}><p>x</p></Dialog>)
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('renders confirm and cancel actions', () => {
    render(
      <Dialog open={true} title="Delete?" onClose={vi.fn()} onConfirm={vi.fn()} confirmLabel="Delete" cancelLabel="Cancel">
        <p>x</p>
      </Dialog>
    )
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
  })

  it('calls onConfirm when confirm button clicked', () => {
    const onConfirm = vi.fn()
    render(
      <Dialog open={true} title="Delete?" onClose={vi.fn()} onConfirm={onConfirm} confirmLabel="Delete">
        <p>x</p>
      </Dialog>
    )
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onConfirm).toHaveBeenCalledOnce()
  })

  it('applies danger variant class to confirm button', () => {
    render(
      <Dialog open={true} title="Delete?" onClose={vi.fn()} onConfirm={vi.fn()} confirmLabel="Delete" danger>
        <p>x</p>
      </Dialog>
    )
    expect(screen.getByRole('button', { name: 'Delete' }).className).toContain('btn-danger')
  })

  it('shows loading state on confirm button', () => {
    render(
      <Dialog open={true} title="Saving" onClose={vi.fn()} onConfirm={vi.fn()} confirmLabel="Save" loading>
        <p>x</p>
      </Dialog>
    )
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })

  it('has aria-labelledby pointing to title', () => {
    render(<Dialog open={true} title="My Dialog" onClose={vi.fn()}><p>x</p></Dialog>)
    const dialog = screen.getByRole('dialog')
    const labelledById = dialog.getAttribute('aria-labelledby')
    expect(labelledById).toBeTruthy()
    expect(document.getElementById(labelledById!)?.textContent).toBe('My Dialog')
  })
})
