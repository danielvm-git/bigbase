import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QuickActions } from './QuickActions'

describe('QuickActions', () => {
  it('shows empty state when no actions', () => {
    render(<QuickActions actions={[]} onAction={() => {}} />)
    expect(screen.getByText('No quick actions')).toBeInTheDocument()
  })

  it('renders action buttons with icons and labels', () => {
    const actions = [
      { icon: '+', label: 'Deploy', link: '/deploy' },
      { icon: '⚡', label: 'Run', link: '/run' },
    ]
    render(<QuickActions actions={actions} onAction={() => {}} />)

    expect(screen.getByText('+ Deploy')).toBeInTheDocument()
    expect(screen.getByText('⚡ Run')).toBeInTheDocument()
  })

  it('calls onAction with link on click', () => {
    const onAction = vi.fn()
    const actions = [{ icon: '+', label: 'Deploy', link: '/deploy' }]
    render(<QuickActions actions={actions} onAction={onAction} />)

    fireEvent.click(screen.getByText('+ Deploy'))
    expect(onAction).toHaveBeenCalledWith('/deploy')
  })

  it('does not call onAction for disabled actions', () => {
    const onAction = vi.fn()
    const actions = [
      { icon: '📦', label: 'Archive', link: '/archive', disabled: true },
      { icon: '+', label: 'Deploy', link: '/deploy' },
    ]
    render(<QuickActions actions={actions} onAction={onAction} />)

    const archiveBtn = screen.getByText('📦 Archive')
    fireEvent.click(archiveBtn)
    expect(onAction).not.toHaveBeenCalled()
  })

  it('renders multiple actions horizontally', () => {
    const actions = [
      { icon: 'A', label: 'Alpha', link: '/a' },
      { icon: 'B', label: 'Beta', link: '/b' },
      { icon: 'C', label: 'Gamma', link: '/c' },
    ]
    const { container } = render(<QuickActions actions={actions} onAction={() => {}} />)

    expect(container.querySelector('[data-testid="quick-actions"]')).toBeInTheDocument()
    expect(screen.getAllByRole('button')).toHaveLength(3)
  })
})
