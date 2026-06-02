import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { RequestChart } from './RequestChart'

describe('RequestChart', () => {
  it('shows "No requests" when total is zero', () => {
    render(<RequestChart byStatus={{}} total={0} />)
    expect(screen.getByText('No requests')).toBeInTheDocument()
  })

  it('renders bar segments for each status code', () => {
    render(<RequestChart byStatus={{ '200': 80, '404': 15, '500': 5 }} total={100} />)

    expect(screen.getByText(/100 total/)).toBeInTheDocument()
    expect(screen.getByTitle('200: 80')).toBeInTheDocument()
    expect(screen.getByTitle('404: 15')).toBeInTheDocument()
    expect(screen.getByTitle('500: 5')).toBeInTheDocument()
  })

  it('renders legend with status codes and counts', () => {
    render(<RequestChart byStatus={{ '200': 50, '500': 3 }} total={53} />)

    expect(screen.getByText('200: 50')).toBeInTheDocument()
    expect(screen.getByText('500: 3')).toBeInTheDocument()
  })

  it('filters bars with less than 1%', () => {
    // 0.5% of 200 = 1 → filtered out
    render(<RequestChart byStatus={{ '200': 199, '404': 1 }} total={200} />)

    expect(screen.getByTitle('200: 199')).toBeInTheDocument()
    expect(screen.queryByTitle('404: 1')).not.toBeInTheDocument()
  })

  it('handles missing status codes gracefully with gray fallback', () => {
    render(<RequestChart byStatus={{ '999': 10 }} total={10} />)

    expect(screen.getByTitle('999: 10')).toBeInTheDocument()
    expect(screen.getByText('999: 10')).toBeInTheDocument()
  })
})
