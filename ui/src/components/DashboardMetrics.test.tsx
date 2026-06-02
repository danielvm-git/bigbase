import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DashboardMetrics } from './DashboardMetrics'

const defaultSystem = {
  cpu_percent: 45.2, memory_mb: 512, goroutines: 42, uptime_seconds: 3600,
}

describe('DashboardMetrics', () => {
  it('renders metric cards when system data is present', () => {
    render(<DashboardMetrics
      system={defaultSystem}
      requests={{ total: 1500, avg_latency_ms: 12, by_status: { '200': 1490, '500': 10 } }}
      componentCount={14}
      healthOk={true}
    />)
    expect(screen.getByText('Request Rate')).toBeInTheDocument()
    expect(screen.getByText('1500')).toBeInTheDocument()
    expect(screen.getByText('45.2%')).toBeInTheDocument()
    expect(screen.getByText(/14 healthy/)).toBeInTheDocument()
  })

  it('shows placeholder when system data is missing', () => {
    render(<DashboardMetrics componentCount={0} healthOk={false} />)
    expect(screen.getByText(/unavailable/i)).toBeInTheDocument()
  })

  it('shows error color when server errors exist', () => {
    render(<DashboardMetrics
      system={defaultSystem}
      requests={{ total: 100, avg_latency_ms: 5, by_status: { '500': 5 } }}
      componentCount={14}
      healthOk={true}
    />)
    const errorCount = screen.getByText('5')
    expect(errorCount.style.color).toBe('var(--error-fg)')
  })

  it('shows success color when no errors', () => {
    render(<DashboardMetrics
      system={defaultSystem}
      requests={{ total: 100, avg_latency_ms: 5, by_status: {} }}
      componentCount={14}
      healthOk={true}
    />)
    const count = screen.getByText('0')
    expect(count.style.color).toBe('var(--success-fg)')
  })

  it('shows components as healthy with healthOk', () => {
    render(<DashboardMetrics
      system={defaultSystem}
      componentCount={16}
      healthOk={true}
    />)
    const comp = screen.getByText(/16 healthy/)
    expect(comp.style.color).toBe('var(--success-fg)')
  })

  it('shows components as warning when not healthOk', () => {
    render(<DashboardMetrics
      system={defaultSystem}
      componentCount={12}
      healthOk={false}
    />)
    const comp = screen.getByText(/12 healthy/)
    expect(comp.style.color).toBe('var(--warning-fg)')
  })

  it('shows CPU bar with correct width', () => {
    render(<DashboardMetrics system={defaultSystem} componentCount={0} healthOk={true} />)
    expect(screen.getByText('45.2%')).toBeInTheDocument()
  })
})
