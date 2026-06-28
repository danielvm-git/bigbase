import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { CodeBlock } from './CodeBlock'

describe('CodeBlock', () => {
  it('renders code content', () => {
    render(<CodeBlock code="const x = 1" />)
    expect(screen.getByText('const x = 1')).toBeInTheDocument()
  })

  it('renders in a pre/code element', () => {
    const { container } = render(<CodeBlock code="hello" />)
    expect(container.querySelector('pre')).toBeInTheDocument()
    expect(container.querySelector('code')).toBeInTheDocument()
  })

  it('renders language label when provided', () => {
    render(<CodeBlock code="const x = 1" language="typescript" />)
    expect(screen.getByText('typescript')).toBeInTheDocument()
  })

  it('renders line numbers when showLineNumbers is set', () => {
    render(<CodeBlock code={"line1\nline2\nline3"} showLineNumbers />)
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('does not render line numbers by default', () => {
    render(<CodeBlock code={"line1\nline2"} />)
    expect(screen.queryByText('1')).not.toBeInTheDocument()
  })

  it('applies maxHeight style', () => {
    const { container } = render(<CodeBlock code="x" maxHeight="200px" />)
    const pre = container.querySelector('pre')
    expect(pre?.style.maxHeight).toBe('200px')
  })

  it('renders title when provided', () => {
    render(<CodeBlock code="x" title="example.ts" />)
    expect(screen.getByText('example.ts')).toBeInTheDocument()
  })
})
