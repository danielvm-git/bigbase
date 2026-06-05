import { useState, useEffect } from 'react'
import { Button } from './Button'

const STORAGE_KEY = 'bigbase_tutorial_step'
const TUTORIAL_DONE_KEY = 'bigbase_tutorial_done'

interface TutorialStep {
  id: string
  title: string
  body: string
  action?: string
  link?: string
}

const STEPS: TutorialStep[] = [
  {
    id: 'welcome',
    title: 'Welcome to BigBase',
    body: 'BigBase is a self-hosted Backend-as-a-Service. Let\'s walk through the basics in under 2 minutes.',
  },
  {
    id: 'create_database',
    title: 'Create your first database',
    body: 'Go to Data Studio and click "New collection" to create a table for your app data. Collections work like database tables with a flexible JSON schema.',
    action: 'Open Data Studio',
    link: '/data',
  },
  {
    id: 'add_collection',
    title: 'Add records to your collection',
    body: 'Click "New record" inside your collection to insert data. You can also use the REST API at /api/collections/:name.',
    action: 'Open Data Studio',
    link: '/data',
  },
  {
    id: 'api_call',
    title: 'Make your first API call',
    body: 'Use the SQL editor to run read-only queries, or call the Collections REST API from your app. Try: GET /api/collections/my-collection',
    action: 'Open SQL Editor',
    link: '/sql',
  },
]

interface TutorialOverlayProps {
  onClose?: () => void
}

export function TutorialOverlay({ onClose }: TutorialOverlayProps) {
  const [stepIndex, setStepIndex] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY)
    return saved ? parseInt(saved, 10) : 0
  })

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, String(stepIndex))
  }, [stepIndex])

  const step = STEPS[stepIndex]
  const isLast = stepIndex === STEPS.length - 1
  const progress = `${stepIndex + 1} / ${STEPS.length}`

  function handleNext() {
    if (isLast) {
      handleDone()
    } else {
      setStepIndex(i => i + 1)
    }
  }

  function handlePrev() {
    setStepIndex(i => Math.max(0, i - 1))
  }

  function handleDone() {
    localStorage.setItem(TUTORIAL_DONE_KEY, 'true')
    localStorage.removeItem(STORAGE_KEY)
    onClose?.()
  }

  return (
    <div className="tutorial-backdrop" role="dialog" aria-modal="true" aria-label="Tutorial" data-testid="tutorial-overlay">
      <div className="tutorial-modal">
        <div className="tutorial-header">
          <span className="tutorial-progress dim">{progress}</span>
          <button type="button" className="btn-icon tutorial-close" onClick={handleDone} aria-label="Close tutorial">
            ×
          </button>
        </div>

        <div className="tutorial-body">
          <h2 className="tutorial-title">{step.title}</h2>
          <p className="tutorial-text">{step.body}</p>
        </div>

        <div className="tutorial-step-dots" aria-hidden>
          {STEPS.map((_, i) => (
            <span key={i} className={`tutorial-dot${i === stepIndex ? ' tutorial-dot--active' : ''}`} />
          ))}
        </div>

        <div className="tutorial-footer">
          {stepIndex > 0 && (
            <Button variant="secondary" size="sm" onClick={handlePrev}>
              Back
            </Button>
          )}
          <div style={{ flex: 1 }} />
          {step.action && step.link && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => { window.location.hash = step.link! }}
            >
              {step.action}
            </Button>
          )}
          <Button variant="primary" size="sm" onClick={handleNext}>
            {isLast ? 'Finish' : 'Next'}
          </Button>
        </div>
      </div>
    </div>
  )
}

// useTutorialTrigger returns show/hide helpers and whether to show the tutorial button.
export function useTutorialTrigger() {
  const [visible, setVisible] = useState(false)
  const isDone = localStorage.getItem(TUTORIAL_DONE_KEY) === 'true'

  return {
    visible,
    isDone,
    open: () => setVisible(true),
    close: () => setVisible(false),
  }
}
