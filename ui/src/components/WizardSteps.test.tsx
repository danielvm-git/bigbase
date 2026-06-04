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
    const { container } = render(<WizardSteps steps={steps} current={2} />)
    // Each step is a .wizard-steps-item <li> with a .wizard-step child div
    // whose className encodes the --active / --done modifier.
    const stepDivs = container.querySelectorAll('.wizard-step')
    expect(stepDivs[1].className).toContain('wizard-step--active')
  })

  it('marks earlier steps with --done class', () => {
    const { container } = render(<WizardSteps steps={steps} current={3} />)
    const stepDivs = container.querySelectorAll('.wizard-step')
    expect(stepDivs[0].className).toContain('wizard-step--done')
    expect(stepDivs[1].className).toContain('wizard-step--done')
    expect(stepDivs[2].className).toContain('wizard-step--active')
  })

  it('renders a check icon for done steps', () => {
    const { container } = render(<WizardSteps steps={steps} current={3} />)
    // Main version renders <Icon name="check" size={14} /> inside .wizard-step-num
    const checkIcons = container.querySelectorAll('.wizard-step-num svg')
    expect(checkIcons.length).toBe(2)
  })

  it('renders number for current step', () => {
    const { container } = render(<WizardSteps steps={steps} current={2} />)
    // Current step's .wizard-step-num should contain the literal "2"
    const numSpans = container.querySelectorAll('.wizard-step-num')
    expect(numSpans[1].textContent).toBe('2')
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
    expect(lines[0].className).toContain('wizard-step-line--done')
    expect(lines[1].className).toContain('wizard-step-line--done')
    expect(lines[2].className).not.toContain('wizard-step-line--done')
  })

  it('exposes aria-label Progress on the list', () => {
    render(<WizardSteps steps={steps} current={1} />)
    expect(screen.getByRole('list', { name: 'Progress' })).toBeInTheDocument()
  })
})
