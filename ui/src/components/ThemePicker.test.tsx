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
    expect(trigger.getAttribute('aria-haspopup')).toBe('menu')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
  })

  it('opens menu on click and sets aria-expanded=true', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    const trigger = screen.getByRole('button', { name: /Accent theme/ })
    fireEvent.click(trigger)
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(screen.getByRole('menu')).toBeInTheDocument()
  })

  it('lists all accent themes as options', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('menuitemradio')
    expect(options.length).toBe(13) // default + 12 months
  })

  it('marks current value as aria-checked=true', () => {
    render(<ThemePicker value="march" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('menuitemradio')
    const selected = options.filter(o => o.getAttribute('aria-checked') === 'true')
    expect(selected).toHaveLength(1)
    expect(selected[0]).toHaveTextContent('March — Purple')
  })

  it('calls onChange with selected id and closes menu', () => {
    const onChange = vi.fn()
    render(<ThemePicker value="default" onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('menuitemradio')
    fireEvent.click(options[1]) // january
    expect(onChange).toHaveBeenCalledWith('january')
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('closes on Escape', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('closes on outside click', () => {
    render(
      <div>
        <ThemePicker value="default" onChange={() => {}} />
        <button type="button" data-testid="outside">Outside</button>
      </div>,
    )
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.mouseDown(screen.getByTestId('outside'))
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('returns focus to trigger after selecting an option (a11y)', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    const trigger = screen.getByRole('button', { name: /Accent theme/ })
    fireEvent.click(trigger)
    const options = screen.getAllByRole('menuitemradio')
    fireEvent.click(options[1]) // january
    expect(document.activeElement).toBe(trigger)
  })

  it('returns focus to trigger after Escape (a11y)', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    const trigger = screen.getByRole('button', { name: /Accent theme/ })
    fireEvent.click(trigger)
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(document.activeElement).toBe(trigger)
  })

  it('returns focus to trigger after outside click (a11y)', () => {
    render(
      <div>
        <ThemePicker value="default" onChange={() => {}} />
        <button type="button" data-testid="outside">Outside</button>
      </div>,
    )
    const trigger = screen.getByRole('button', { name: /Accent theme/ })
    fireEvent.click(trigger)
    fireEvent.mouseDown(screen.getByTestId('outside'))
    expect(document.activeElement).toBe(trigger)
  })

  it('moves focus to the selected item when the menu opens', () => {
    render(<ThemePicker value="march" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('menuitemradio')
    const selected = options.find(o => o.getAttribute('aria-checked') === 'true')
    expect(document.activeElement).toBe(selected)
  })

  it('navigates with ArrowDown/ArrowUp/Home/End (roving focus, wrapping)', () => {
    render(<ThemePicker value="default" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const options = screen.getAllByRole('menuitemradio')
    const menu = screen.getByRole('menu')
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(options[1])
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(options[2])
    fireEvent.keyDown(menu, { key: 'ArrowUp' })
    expect(document.activeElement).toBe(options[1])
    fireEvent.keyDown(menu, { key: 'Home' })
    expect(document.activeElement).toBe(options[0])
    fireEvent.keyDown(menu, { key: 'End' })
    expect(document.activeElement).toBe(options[12])
    fireEvent.keyDown(menu, { key: 'ArrowDown' }) // wraps past the end
    expect(document.activeElement).toBe(options[0])
  })

  it('selects the focused item with Enter and closes the menu', () => {
    const onChange = vi.fn()
    render(<ThemePicker value="default" onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const menu = screen.getByRole('menu')
    fireEvent.keyDown(menu, { key: 'ArrowDown' }) // january
    fireEvent.keyDown(menu, { key: 'Enter' })
    expect(onChange).toHaveBeenCalledWith('january')
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('closes the menu when focus leaves it (Tab-out)', () => {
    render(
      <div>
        <ThemePicker value="default" onChange={() => {}} />
        <button type="button" data-testid="after">After</button>
      </div>,
    )
    fireEvent.click(screen.getByRole('button', { name: /Accent theme/ }))
    const menu = screen.getByRole('menu')
    fireEvent.blur(menu, { relatedTarget: screen.getByTestId('after') })
    expect(screen.queryByRole('menu')).toBeNull()
  })
})
