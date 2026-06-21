import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StreamLog } from './StreamLog'

describe('StreamLog', () => {
  it('renders log lines with line numbers', () => {
    render(<StreamLog logs={['Line one', 'Line two']} />)

    expect(screen.getByText('Line one')).toBeInTheDocument()
    expect(screen.getByText('Line two')).toBeInTheDocument()
    expect(screen.getAllByTestId('stream-log-line')).toHaveLength(2)
  })

  it('shows "No logs" when logs array is empty', () => {
    render(<StreamLog logs={[]} />)

    expect(screen.getByTestId('stream-log-empty')).toBeInTheDocument()
    expect(screen.getByText('No logs')).toBeInTheDocument()
  })

  it('shows animated cursor when isStreaming is true', () => {
    render(<StreamLog logs={['building...']} isStreaming={true} />)

    expect(screen.getByTestId('stream-log-cursor')).toBeInTheDocument()
    expect(screen.getByText('▊')).toBeInTheDocument()
  })

  it('does not show cursor when isStreaming is false', () => {
    render(<StreamLog logs={['done']} isStreaming={false} />)

    expect(screen.queryByTestId('stream-log-cursor')).not.toBeInTheDocument()
  })

  it('shows cursor even with empty logs when streaming', () => {
    render(<StreamLog logs={[]} isStreaming={true} />)

    expect(screen.getByTestId('stream-log-cursor')).toBeInTheDocument()
    expect(screen.queryByTestId('stream-log-empty')).not.toBeInTheDocument()
  })

  it('renders with custom className', () => {
    render(<StreamLog logs={['test']} className="custom-log" />)

    const container = screen.getByTestId('stream-log-container')
    expect(container.className).toContain('custom-log')
  })
})
