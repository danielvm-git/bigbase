import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { WizardSteps } from './WizardSteps'

describe('WizardSteps', () => {
  const steps = ['Source', 'Configure', 'Review', 'Deploy']

  it('renders all step labels', () => {
    render(<WizardSteps steps={steps} current={1} />)
    expect(screen.getByText('Source')).toBeInTheDocument()
    expect(screen.getByText('Configure')).toBeInTheDocument()
    expect(screen.getByText('Review')).toBeInTheDocument()
    expect(screen.getByText('Deploy')).toBeInTheDocument()
  })

  it('marks current step with --active class', () => {
    render(<WizardSteps steps={steps} current={2} />)
    const items = screen.getAllByRole('listitem')
    expect(items[1].className).toContain('wizard-step--active')
  })

  it('marks earlier steps with --done class', () => {
    render(<WizardSteps steps={steps} current={3} />)
    const items = screen.getAllByRole('listitem')
    expect(items[0].className).toContain('wizard-step--done')
    expect(items[1].className).toContain('wizard-step--done')
    expect(items[2].className).toContain('wizard-step--active')
  })

  it('renders a check glyph for done steps', () => {
    render(<WizardSteps steps={steps} current={3} />)
    expect(screen.getAllByText('✓').length).toBe(2)
  })

  it('renders number for current step', () => {
    render(<WizardSteps steps={steps} current={2} />)
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('renders connector lines between every adjacent step pair', () => {
    const { container } = render(<WizardSteps steps={steps} current={1} />)
    // 4 steps → 3 connector lines
    const lines = container.querySelectorAll('.wizard-step-line')
    expect(lines.length).toBe(3)
  })

  it('marks connector line as done when source step is past', () => {
    const { container } = render(<WizardSteps steps={steps} current={3} />)
    const lines = container.querySelectorAll('.wizard-step-line')
    expect(lines[0].className).toContain('done')
    expect(lines[1].className).toContain('done')
    expect(lines[2].className).not.toContain('done')
  })

  it('exposes aria-label Progress on the list', () => {
    render(<WizardSteps steps={steps} current={1} />)
    expect(screen.getByRole('list', { name: 'Progress' })).toBeInTheDocument()
  })
})
