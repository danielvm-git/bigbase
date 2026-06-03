import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemePicker } from './ThemePicker'

describe('ThemePicker', () => {

  it('renders current selection as trigger label', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    expect(screen.getByRole('button', { name: /Accent theme/ })).toHaveTextContent(
      'Indigo (default)',
    )
  })

  it('trigger has aria-haspopup and aria-expanded=false initially', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    const trigger = screen.getByRole('button', { name: /Accent theme/ })
    expect(trigger.getAttribute('aria-haspopup')).toBe('listbox')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
  })

  it('opens menu on click and sets aria-expanded=true', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    const trigger = screen.getByRole('button', { name: /Accent theme/ })
    fireEvent.click(trigger)
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(screen.getByRole('listbox')).toBeInTheDocument()
  })

  it('lists all accent themes as options', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('option')
    expect(options.length).toBe(13) // default + 12 months
  })

  it('marks current value as aria-selected=true', () => {
    render(<ThemePicker value="march" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('option')
    const selected = options.filter(o => o.getAttribute('aria-selected') === 'true')
    expect(selected).toHaveLength(1)
    expect(selected[0]).toHaveTextContent('March — Purple')
  })

  it('calls onChange with selected id and closes menu', () => {
    const onChange = vi.fn()
    render(<ThemePicker value="default" onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('option')
    fireEvent.click(options[1]) // january
    expect(onChange).toHaveBeenCalledWith('january')
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('closes on Escape', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('closes on outside click', () => {
    render(
      <div>
        <ThemePicker value="default" onChange={() => {}} />
        <button type="button" data-testid="outside">Outside</button>
      </div>,
    )
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    fireEvent.mouseDown(screen.getByTestId('outside'))
    expect(screen.queryByRole('listbox')).toBeNull()
  })
})
