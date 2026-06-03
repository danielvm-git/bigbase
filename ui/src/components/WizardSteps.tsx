import { Fragment } from 'react'

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
        const lineDone = done
        return (
          <Fragment key={label}>
            <li
              className={[
                'wizard-step',
                done ? 'wizard-step--done' : '',
                active ? 'wizard-step--active' : '',
              ].filter(Boolean).join(' ')}
            >
              <span className="wizard-step-num">{done ? '✓' : n}</span>
              <span className="wizard-step-label">{label}</span>
            </li>
            {i < steps.length - 1 && (
              <div
                className={`wizard-step-line${lineDone ? ' done' : ''}`}
                aria-hidden="true"
              />
            )}
          </Fragment>
        )
      })}
    </ol>
  )
}
