import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TagInput } from './TagInput'

describe('TagInput', () => {
  it('renders input', () => {
    render(<TagInput tags={[]} onChange={vi.fn()} />)
    expect(screen.getByRole('textbox')).toBeInTheDocument()
  })

  it('renders existing tags', () => {
    render(<TagInput tags={['react', 'typescript']} onChange={vi.fn()} />)
    expect(screen.getByText('react')).toBeInTheDocument()
    expect(screen.getByText('typescript')).toBeInTheDocument()
  })

  it('adds tag on Enter', () => {
    const onChange = vi.fn()
    render(<TagInput tags={[]} onChange={onChange} />)
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'newtag' } })
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' })
    expect(onChange).toHaveBeenCalledWith(['newtag'])
  })

  it('adds tag on comma', () => {
    const onChange = vi.fn()
    render(<TagInput tags={[]} onChange={onChange} />)
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'newtag' } })
    fireEvent.keyDown(screen.getByRole('textbox'), { key: ',' })
    expect(onChange).toHaveBeenCalledWith(['newtag'])
  })

  it('removes tag when remove button clicked', () => {
    const onChange = vi.fn()
    render(<TagInput tags={['react', 'ts']} onChange={onChange} />)
    fireEvent.click(screen.getAllByRole('button', { name: /remove/i })[0])
    expect(onChange).toHaveBeenCalledWith(['ts'])
  })

  it('does not add duplicate tags', () => {
    const onChange = vi.fn()
    render(<TagInput tags={['react']} onChange={onChange} />)
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'react' } })
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' })
    expect(onChange).not.toHaveBeenCalled()
  })

  it('respects maxTags limit', () => {
    const onChange = vi.fn()
    render(<TagInput tags={['a', 'b', 'c']} onChange={onChange} maxTags={3} />)
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'new' } })
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' })
    expect(onChange).not.toHaveBeenCalled()
  })

  it('removes last tag on Backspace when input is empty', () => {
    const onChange = vi.fn()
    render(<TagInput tags={['react', 'ts']} onChange={onChange} />)
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Backspace' })
    expect(onChange).toHaveBeenCalledWith(['react'])
  })

  it('shows placeholder when no tags', () => {
    render(<TagInput tags={[]} onChange={vi.fn()} placeholder="Add tags" />)
    expect(screen.getByPlaceholderText('Add tags')).toBeInTheDocument()
  })
})
