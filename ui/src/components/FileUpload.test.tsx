import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { FileUpload } from './FileUpload'

describe('FileUpload', () => {
  it('renders drop zone', () => {
    render(<FileUpload onFiles={vi.fn()} />)
    expect(screen.getByRole('button', { name: /upload|choose|browse/i })).toBeInTheDocument()
  })

  it('renders label text', () => {
    render(<FileUpload onFiles={vi.fn()} label="Upload files" />)
    expect(screen.getByText('Upload files')).toBeInTheDocument()
  })

  it('renders hidden file input', () => {
    const { container } = render(<FileUpload onFiles={vi.fn()} />)
    expect(container.querySelector('input[type="file"]')).toBeInTheDocument()
  })

  it('shows accepted types', () => {
    render(<FileUpload onFiles={vi.fn()} accept=".png,.jpg" />)
    expect(screen.getByText(/\.png|\.jpg/)).toBeInTheDocument()
  })

  it('shows max size hint', () => {
    render(<FileUpload onFiles={vi.fn()} maxSizeMb={5} />)
    expect(screen.getByText(/5\s*MB/)).toBeInTheDocument()
  })

  it('calls onFiles when files selected', () => {
    const onFiles = vi.fn()
    const { container } = render(<FileUpload onFiles={onFiles} />)
    const input = container.querySelector('input[type="file"]')!
    const file = new File(['content'], 'test.txt', { type: 'text/plain' })
    fireEvent.change(input, { target: { files: [file] } })
    expect(onFiles).toHaveBeenCalledWith([file])
  })

  it('shows error for oversized file', () => {
    const onFiles = vi.fn()
    const { container } = render(<FileUpload onFiles={onFiles} maxSizeMb={0.001} />)
    const input = container.querySelector('input[type="file"]')!
    const bigFile = new File(['x'.repeat(2000)], 'big.txt', { type: 'text/plain' })
    Object.defineProperty(bigFile, 'size', { value: 2000 })
    fireEvent.change(input, { target: { files: [bigFile] } })
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })

  it('renders multiple attribute when multiple prop set', () => {
    const { container } = render(<FileUpload onFiles={vi.fn()} multiple />)
    expect(container.querySelector('input[type="file"]')).toHaveAttribute('multiple')
  })
})
