import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SystemStatusPanel } from './SystemStatusPanel'

const system = {
  cpu_percent: 23.5,
  memory_mb: 512,
  goroutines: 42,
  uptime_seconds: 3600 * 14 + 3600 * 6,
}

describe('SystemStatusPanel', () => {
  it('shows operational banner and healthy badge', () => {
    render(
      <SystemStatusPanel
        healthOk
        componentCount={16}
        system={system}
        activity={[]}
      />,
    )
    expect(screen.getByText('All systems operational')).toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
    expect(screen.getByText(/16 components/)).toBeInTheDocument()
  })

  it('shows components fraction and CPU percent with bar', () => {
    render(
      <SystemStatusPanel
        healthOk
        componentCount={16}
        system={system}
        activity={[]}
      />,
    )
    expect(screen.getByText('Components')).toBeInTheDocument()
    expect(screen.getByText('all running')).toBeInTheDocument()
    expect(screen.getByText('CPU')).toBeInTheDocument()
    expect(screen.getByText('23.5')).toBeInTheDocument()
    const cpuTile = screen.getByText('CPU').closest('.metric-tile')
    const bar = cpuTile?.querySelector('.bar-fill') as HTMLElement | null
    expect(bar?.style.width).toBe('23.5%')
  })

  it('shows memory in MB with bar', () => {
    render(
      <SystemStatusPanel
        healthOk
        componentCount={16}
        system={system}
        activity={[]}
      />,
    )
    expect(screen.getByText('Memory')).toBeInTheDocument()
    expect(screen.getByText('512 MB')).toBeInTheDocument()
  })

  it('shows degraded copy when not healthy', () => {
    render(
      <SystemStatusPanel
        healthOk={false}
        componentCount={16}
        runningCount={12}
        system={system}
        activity={[]}
      />,
    )
    expect(screen.getByText('System issues detected')).toBeInTheDocument()
    expect(screen.getByText('some components offline')).toBeInTheDocument()
    expect(screen.getByText('12')).toBeInTheDocument()
  })

  it('renders activity rows', () => {
    render(
      <SystemStatusPanel
        healthOk
        componentCount={16}
        system={system}
        activity={[
          { id: '1', label: 'Deploy web-app', when: '2m ago', status: 'ready', statusLabel: 'COMPLETED' },
        ]}
      />,
    )
    expect(screen.getByText('Deploy web-app')).toBeInTheDocument()
    expect(screen.getByText('COMPLETED')).toBeInTheDocument()
  })
})
