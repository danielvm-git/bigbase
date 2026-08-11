import { useEffect, useState } from 'react'
import { Card, CardHeader } from './Card'

interface OnboardingStep {
  id: string
  label: string
  done: boolean
}

export function OnboardingChecklist() {
  const [steps, setSteps] = useState<OnboardingStep[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const ctrl = new AbortController()
    fetch('/api/onboarding', { signal: ctrl.signal })
      .then(r => r.ok ? r.json() : { steps: [] })
      .then(data => {
        if (!ctrl.signal.aborted) {
          setSteps(data.steps ?? [])
        }
      })
      .catch(() => {
        if (!ctrl.signal.aborted) setSteps([])
      })
      .finally(() => {
        if (!ctrl.signal.aborted) setLoading(false)
      })
    return () => ctrl.abort()
  }, [])

  const doneCount = steps.filter(s => s.done).length
  const allDone = steps.length > 0 && doneCount === steps.length

  if (allDone) return null

  return (
    <Card data-testid="onboarding-checklist">
      <CardHeader title="Get started">
        <span className="dim" style={{ fontSize: '0.8rem' }} aria-live="polite">
          {loading ? '…' : `${doneCount} / ${steps.length}`}
        </span>
      </CardHeader>

      {loading ? (
        <p className="dim" role="status" aria-busy="true">Loading checklist…</p>
      ) : (
        <ul className="onboarding-list" role="list">
          {steps.map(step => (
            <li
              key={step.id}
              className={`onboarding-item${step.done ? ' onboarding-item--done' : ''}`}
              role="checkbox"
              aria-checked={step.done}
              aria-disabled="true"
              aria-label={`${step.label}, ${step.done ? 'done' : 'to do'}`}
            >
              <span className="onboarding-check" aria-hidden="true">
                {step.done ? '✓' : '○'}
              </span>
              <span className={step.done ? 'dim' : ''}>{step.label}</span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}
