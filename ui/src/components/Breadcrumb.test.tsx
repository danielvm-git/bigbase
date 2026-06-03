import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Breadcrumb } from './Breadcrumb'

describe('Breadcrumb', () => {
  it('renders links and current page with aria-current', () => {
    render(
      <MemoryRouter>
        <Breadcrumb items={[
          { label: 'Functions', to: '/functions' },
          { label: 'my-fn' },
        ]} />
      </MemoryRouter>,
    )
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Functions' })).toHaveAttribute('href', '/functions')
    expect(screen.getByText('my-fn')).toHaveAttribute('aria-current', 'page')
  })
})
