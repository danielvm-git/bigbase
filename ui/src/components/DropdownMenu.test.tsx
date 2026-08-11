import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { DropdownMenu } from './DropdownMenu'

const items = [
  { id: 'edit', label: 'Edit' },
  { id: 'copy', label: 'Copy' },
  { id: 'delete', label: 'Delete', danger: true },
]

describe('DropdownMenu', () => {
  it('renders trigger', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    expect(screen.getByRole('button', { name: 'Options' })).toBeInTheDocument()
  })

  it('menu is closed initially', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('opens menu on trigger click', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    fireEvent.click(screen.getByRole('button', { name: 'Options' }))
    expect(screen.getByRole('menu')).toBeInTheDocument()
  })

  it('makes a non-interactive trigger keyboard-accessible (Enter/Space open menu)', async () => {
    const user = userEvent.setup()
    render(<DropdownMenu trigger={<span>Options</span>} items={items} />)
    const trigger = screen.getByRole('button', { name: 'Options' })
    expect(trigger.tagName).toBe('SPAN')
    expect(trigger).toHaveAttribute('tabindex', '0')
    trigger.focus()
    await user.keyboard('{Enter}')
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    await user.keyboard(' ')
    expect(screen.getByRole('menu')).toBeInTheDocument()
  })

  it('does not double up keyboard controls when trigger is already a button', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(1)
    expect(buttons[0].tagName).toBe('BUTTON')
  })

  it('renders all menu items', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    fireEvent.click(screen.getByRole('button', { name: 'Options' }))
    expect(screen.getAllByRole('menuitem')).toHaveLength(3)
  })

  it('closes on item click', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    fireEvent.click(screen.getByRole('button', { name: 'Options' }))
    fireEvent.click(screen.getByRole('menuitem', { name: 'Edit' }))
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('calls onSelect with item id', () => {
    const onSelect = vi.fn()
    render(<DropdownMenu trigger={<button>Options</button>} items={items} onSelect={onSelect} />)
    fireEvent.click(screen.getByRole('button', { name: 'Options' }))
    fireEvent.click(screen.getByRole('menuitem', { name: 'Edit' }))
    expect(onSelect).toHaveBeenCalledWith('edit')
  })

  it('closes on Escape', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    fireEvent.click(screen.getByRole('button', { name: 'Options' }))
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('applies danger class to danger items', () => {
    render(<DropdownMenu trigger={<button>Options</button>} items={items} />)
    fireEvent.click(screen.getByRole('button', { name: 'Options' }))
    expect(screen.getByRole('menuitem', { name: 'Delete' }).className).toContain('danger')
  })

  it('renders divider', () => {
    const withDivider = [
      { id: 'edit', label: 'Edit' },
      { id: 'divider-1', divider: true as const },
      { id: 'delete', label: 'Delete' },
    ]
    render(<DropdownMenu trigger={<button>Options</button>} items={withDivider} />)
    fireEvent.click(screen.getByRole('button', { name: 'Options' }))
    expect(document.querySelector('.dropdown-divider')).toBeInTheDocument()
  })
})
