import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { JsonTree } from './JsonTree'

describe('JsonTree', () => {
  it('renders a string value', () => {
    render(<JsonTree data="hello" />)
    expect(screen.getByText('"hello"')).toBeInTheDocument()
  })

  it('renders a number value', () => {
    render(<JsonTree data={42} />)
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('renders null', () => {
    render(<JsonTree data={null} />)
    expect(screen.getByText('null')).toBeInTheDocument()
  })

  it('renders boolean', () => {
    render(<JsonTree data={true} />)
    expect(screen.getByText('true')).toBeInTheDocument()
  })

  it('renders object keys', () => {
    render(<JsonTree data={{ name: 'Alice', age: 30 }} />)
    expect(screen.getByText('name')).toBeInTheDocument()
    expect(screen.getByText('age')).toBeInTheDocument()
  })

  it('renders object values', () => {
    render(<JsonTree data={{ name: 'Alice' }} />)
    expect(screen.getByText('"Alice"')).toBeInTheDocument()
  })

  it('renders array items', () => {
    render(<JsonTree data={[1, 2, 3]} />)
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('collapses and expands object on click', () => {
    render(<JsonTree data={{ key: 'value' }} />)
    const toggle = screen.getByRole('button', { name: /collapse|expand/i })
    expect(screen.getByText('"value"')).toBeInTheDocument()
    fireEvent.click(toggle)
    expect(screen.queryByText('"value"')).not.toBeInTheDocument()
    fireEvent.click(toggle)
    expect(screen.getByText('"value"')).toBeInTheDocument()
  })

  it('respects maxDepth by not rendering children beyond limit', () => {
    const deep = { a: { b: { c: 'deep' } } }
    render(<JsonTree data={deep} maxDepth={1} />)
    expect(screen.queryByText('"deep"')).not.toBeInTheDocument()
  })
})
