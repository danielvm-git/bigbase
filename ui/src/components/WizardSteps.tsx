import { Icon } from './Icon'

interface WizardStepsProps {
  steps: string[]
  current: number
}

export function WizardSteps({ steps, current }: WizardStepsProps) {
  return (
    <ol className="wizard-steps" aria-label="Progress">
      {steps.map((label, i) => {
        const n = i + 1
        const done = n < current
        const active = n === current
        const isLast = i === steps.length - 1
        return (
          <li key={label} className="wizard-steps-item" style={{ display: 'contents' }}>
            <div
              className={[
                'wizard-step',
                done ? 'wizard-step--done' : '',
                active ? 'wizard-step--active' : '',
              ].filter(Boolean).join(' ')}
              aria-current={active ? 'step' : undefined}
            >
              <span className="wizard-step-num">
                {done ? <Icon name="check" size={14} /> : n}
              </span>
              <span className="wizard-step-label">{label}</span>
            </div>
            {!isLast && (
              <span
                className={['wizard-step-line', done ? 'wizard-step-line--done' : ''].filter(Boolean).join(' ')}
                aria-hidden
              />
            )}
          </li>
        )
      })}
    </ol>
  )
}
