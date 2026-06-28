import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { Select } from './Select'

const options = [
  { value: 'a', label: 'Option A' },
  { value: 'b', label: 'Option B' },
  { value: 'c', label: 'Option C' },
]

describe('Select', () => {
  it('renders a select element', () => {
    render(<Select options={options} label="Choose" />)
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('renders all options', () => {
    render(<Select options={options} label="Choose" />)
    expect(screen.getAllByRole('option')).toHaveLength(3)
  })

  it('associates label with select', () => {
    render(<Select options={options} label="My Label" />)
    expect(screen.getByLabelText('My Label')).toBeInTheDocument()
  })

  it('shows selected value', () => {
    render(<Select options={options} label="Choose" value="b" onChange={vi.fn()} />)
    expect(screen.getByRole('combobox')).toHaveValue('b')
  })

  it('calls onChange on selection', () => {
    const onChange = vi.fn()
    render(<Select options={options} label="Choose" onChange={onChange} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'b' } })
    expect(onChange).toHaveBeenCalledOnce()
  })

  it('shows error message', () => {
    render(<Select options={options} label="Choose" error="Required" />)
    expect(screen.getByText('Required')).toBeInTheDocument()
  })

  it('shows hint text', () => {
    render(<Select options={options} label="Choose" hint="Pick one" />)
    expect(screen.getByText('Pick one')).toBeInTheDocument()
  })

  it('renders disabled state', () => {
    render(<Select options={options} label="Choose" disabled />)
    expect(screen.getByRole('combobox')).toBeDisabled()
  })

  it('renders placeholder option when provided', () => {
    render(<Select options={options} label="Choose" placeholder="Select..." />)
    expect(screen.getByText('Select...')).toBeInTheDocument()
  })
})
