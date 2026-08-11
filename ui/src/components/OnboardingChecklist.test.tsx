import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { OnboardingChecklist } from './OnboardingChecklist'

const steps = [
  { id: 's1', label: 'Create a site', done: true },
  { id: 's2', label: 'Deploy your app', done: false },
]

afterEach(() => {
  vi.unstubAllGlobals()
})

function mockFetch(data: { steps: typeof steps } | { steps: [] } = { steps }) {
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve(data) })))
}

describe('OnboardingChecklist', () => {
  it('renders each step as a disabled checkbox with aria-checked state', async () => {
    mockFetch()
    render(<OnboardingChecklist />)
    const checkboxes = await screen.findAllByRole('checkbox')
    expect(checkboxes).toHaveLength(2)
    expect(checkboxes[0]).toHaveAttribute('aria-checked', 'true')
    expect(checkboxes[1]).toHaveAttribute('aria-checked', 'false')
    expect(checkboxes[0]).toHaveAttribute('aria-disabled', 'true')
  })

  it('gives each step an accessible label including done/to do state', async () => {
    mockFetch()
    render(<OnboardingChecklist />)
    expect(await screen.findByRole('checkbox', { name: 'Create a site, done' })).toBeInTheDocument()
    expect(await screen.findByRole('checkbox', { name: 'Deploy your app, to do' })).toBeInTheDocument()
  })

  it('keeps the check glyph purely decorative (aria-hidden)', async () => {
    mockFetch()
    render(<OnboardingChecklist />)
    await screen.findAllByRole('checkbox')
    expect(screen.getByText('✓')).toHaveAttribute('aria-hidden', 'true')
    expect(screen.getByText('○')).toHaveAttribute('aria-hidden', 'true')
  })

  it('announces the progress count via aria-live', async () => {
    mockFetch()
    render(<OnboardingChecklist />)
    await screen.findAllByRole('checkbox')
    const progress = screen.getByText('1 / 2')
    expect(progress).toHaveAttribute('aria-live', 'polite')
  })

  it('renders nothing when all steps are done', async () => {
    mockFetch({ steps: [{ id: 's1', label: 'Create a site', done: true }] })
    const { container } = render(<OnboardingChecklist />)
    await waitFor(() => expect(container).toBeEmptyDOMElement())
  })
})
