import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { Checkbox } from './Checkbox'

describe('Checkbox', () => {
  it('renders with a label', () => {
    render(<Checkbox label="Accept terms" />)
    expect(screen.getByLabelText('Accept terms')).toBeInTheDocument()
  })

  it('renders unchecked by default', () => {
    render(<Checkbox label="Check me" />)
    expect(screen.getByRole('checkbox')).not.toBeChecked()
  })

  it('renders checked when defaultChecked is set', () => {
    render(<Checkbox label="Check me" defaultChecked />)
    expect(screen.getByRole('checkbox')).toBeChecked()
  })

  it('works as controlled component', () => {
    const onChange = vi.fn()
    render(<Checkbox label="Controlled" checked={true} onChange={onChange} />)
    expect(screen.getByRole('checkbox')).toBeChecked()
  })

  it('calls onChange when clicked', () => {
    const onChange = vi.fn()
    render(<Checkbox label="Click me" onChange={onChange} />)
    fireEvent.click(screen.getByRole('checkbox'))
    expect(onChange).toHaveBeenCalledOnce()
  })

  it('renders disabled state', () => {
    render(<Checkbox label="Disabled" disabled />)
    expect(screen.getByRole('checkbox')).toBeDisabled()
  })

  it('applies indeterminate state', () => {
    render(<Checkbox label="Indeterminate" indeterminate />)
    const cb = screen.getByRole('checkbox') as HTMLInputElement
    expect(cb.indeterminate).toBe(true)
  })

  it('shows error message', () => {
    render(<Checkbox label="Required" error="This field is required" />)
    expect(screen.getByText('This field is required')).toBeInTheDocument()
  })
})
