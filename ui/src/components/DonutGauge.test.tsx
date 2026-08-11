import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { DonutGauge } from './DonutGauge'

describe('DonutGauge', () => {
  it('has an accessible name including the data', () => {
    render(<DonutGauge used={30} total={100} label="CPU" />)
    expect(screen.getByRole('img', { name: 'CPU: 30 of 100 (30 percent)' })).toBeInTheDocument()
  })

  it('defaults the name to gauge with the data', () => {
    render(<DonutGauge used={512} total={1024} />)
    expect(screen.getByRole('img', { name: 'gauge: 512 of 1024 (50 percent)' })).toBeInTheDocument()
  })

  it('clamps the percentage to 100', () => {
    render(<DonutGauge used={200} total={100} label="Disk" />)
    expect(screen.getByRole('img', { name: 'Disk: 200 of 100 (100 percent)' })).toBeInTheDocument()
  })
})
