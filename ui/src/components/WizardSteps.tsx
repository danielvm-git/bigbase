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
        return (
          <li
            key={label}
            className={[
              'wizard-step',
              done ? 'wizard-step--done' : '',
              active ? 'wizard-step--active' : '',
            ].filter(Boolean).join(' ')}
          >
            <span className="wizard-step-num">{done ? '✓' : n}</span>
            <span className="wizard-step-label">{label}</span>
          </li>
        )
      })}
    </ol>
  )
}
