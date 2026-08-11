import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { EnvVarEditor } from './EnvVarEditor'

// The editor is controlled, so a stateful wrapper mirrors real usage.
function StatefulEnvVarEditor({ initial }: { initial: Record<string, string> }) {
  const [vars, setVars] = useState(initial)
  return <EnvVarEditor vars={vars} onChange={setVars} />
}

describe('EnvVarEditor', () => {
  it('renders existing key-value pairs', () => {
    const vars = { NODE_ENV: 'production', PORT: '3000' }
    render(<EnvVarEditor vars={vars} onChange={() => {}} />)

    expect(screen.getByDisplayValue('NODE_ENV')).toBeInTheDocument()
    expect(screen.getByDisplayValue('production')).toBeInTheDocument()
    expect(screen.getByDisplayValue('PORT')).toBeInTheDocument()
    expect(screen.getByDisplayValue('3000')).toBeInTheDocument()
  })

  it('shows empty state when no vars', () => {
    render(<EnvVarEditor vars={{}} onChange={() => {}} />)

    expect(screen.getByTestId('env-var-editor-empty')).toBeInTheDocument()
    expect(screen.getByText('No environment variables configured')).toBeInTheDocument()
  })

  it('fires onChange when a value is edited', () => {
    const onChange = vi.fn()
    render(<EnvVarEditor vars={{ KEY: 'old' }} onChange={onChange} />)

    const valueInput = screen.getByDisplayValue('old')
    fireEvent.change(valueInput, { target: { value: 'new' } })

    expect(onChange).toHaveBeenCalledWith({ KEY: 'new' })
  })

  it('fires onChange when a key is edited', () => {
    const onChange = vi.fn()
    render(<EnvVarEditor vars={{ OLD_KEY: 'val' }} onChange={onChange} />)

    const keyInput = screen.getByDisplayValue('OLD_KEY')
    fireEvent.change(keyInput, { target: { value: 'NEW_KEY' } })

    expect(onChange).toHaveBeenCalledWith({ NEW_KEY: 'val' })
  })

  it('adds a new variable row on Add click', () => {
    const onChange = vi.fn()
    render(<EnvVarEditor vars={{ FOO: 'bar' }} onChange={onChange} />)

    fireEvent.click(screen.getByText('Add Variable'))

    expect(onChange).toHaveBeenCalled()
    const updated = onChange.mock.calls[0][0] as Record<string, string>
    expect(Object.keys(updated)).toHaveLength(2)
    expect(updated.FOO).toBe('bar')
  })

  it('removes a variable row on Remove click', () => {
    const onChange = vi.fn()
    render(<EnvVarEditor vars={{ FOO: 'bar', BAZ: 'qux' }} onChange={onChange} />)

    const removeBtns = screen.getAllByText('Remove')
    fireEvent.click(removeBtns[0])

    expect(onChange).toHaveBeenCalled()
    const updated = onChange.mock.calls[0][0] as Record<string, string>
    expect(updated).not.toHaveProperty('FOO')
    expect(updated.BAZ).toBe('qux')
  })

  it('adds variable from empty state via Add Variable button', () => {
    const onChange = vi.fn()
    render(<EnvVarEditor vars={{}} onChange={onChange} />)

    fireEvent.click(screen.getByText('Add Variable'))

    expect(onChange).toHaveBeenCalled()
    const updated = onChange.mock.calls[0][0] as Record<string, string>
    expect(Object.keys(updated)).toHaveLength(1)
    expect(Object.keys(updated)[0]).toMatch(/^KEY_/)
  })

  it('does not overwrite existing key when renaming to a collision', () => {
    const onChange = vi.fn()
    render(<EnvVarEditor vars={{ FOO: 'fooVal', BAR: 'barVal' }} onChange={onChange} />)

    const fooInput = screen.getByDisplayValue('FOO')
    fireEvent.change(fooInput, { target: { value: 'BAR' } })

    // onChange should NOT be called because BAR already exists
    expect(onChange).not.toHaveBeenCalled()
  })

  it('does not show Revert for untouched rows', () => {
    render(<StatefulEnvVarEditor initial={{ FOO: 'bar', BAZ: 'qux' }} />)

    expect(screen.queryByText('Revert')).not.toBeInTheDocument()
  })

  it('shows a Revert action for an edited row and restores the original value', () => {
    render(<StatefulEnvVarEditor initial={{ KEY: 'old' }} />)

    fireEvent.change(screen.getByDisplayValue('old'), { target: { value: 'new' } })
    expect(screen.getByDisplayValue('new')).toBeInTheDocument()
    expect(screen.getByText('Revert')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Revert'))

    expect(screen.getByDisplayValue('old')).toBeInTheDocument()
    expect(screen.queryByText('Revert')).not.toBeInTheDocument()
  })

  it('Revert restores the original key after a rename', () => {
    render(<StatefulEnvVarEditor initial={{ OLD_KEY: 'val' }} />)

    fireEvent.change(screen.getByDisplayValue('OLD_KEY'), { target: { value: 'NEW_KEY' } })
    expect(screen.getByDisplayValue('NEW_KEY')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Revert'))

    expect(screen.getByDisplayValue('OLD_KEY')).toBeInTheDocument()
    expect(screen.getByDisplayValue('val')).toBeInTheDocument()
    expect(screen.queryByText('Revert')).not.toBeInTheDocument()
  })

  it('Revert removes a row that was added during editing', () => {
    render(<StatefulEnvVarEditor initial={{ FOO: 'bar' }} />)

    fireEvent.click(screen.getByText('Add Variable'))
    expect(screen.getByDisplayValue('KEY_1')).toBeInTheDocument()

    fireEvent.change(screen.getByDisplayValue(''), { target: { value: 'secret' } })
    fireEvent.click(screen.getByText('Revert'))

    expect(screen.queryByDisplayValue('KEY_1')).not.toBeInTheDocument()
    expect(screen.getByDisplayValue('bar')).toBeInTheDocument()
  })
})
