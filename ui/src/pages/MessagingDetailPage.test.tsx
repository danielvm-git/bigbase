import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import MessagingDetailPage from './MessagingDetailPage'

describe('MessagingDetailPage', () => {
  it('renders template editor for known template', async () => {
    render(
      <MemoryRouter initialEntries={['/messaging/tpl-welcome']}>
        <Routes>
          <Route path="/messaging/:id" element={<MessagingDetailPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Welcome email' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Editor' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Preview' })).toBeInTheDocument()
    })
  })

  it('shows not found for unknown template', () => {
    render(
      <MemoryRouter initialEntries={['/messaging/unknown']}>
        <Routes>
          <Route path="/messaging/:id" element={<MessagingDetailPage />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('Template not found')).toBeInTheDocument()
  })
})
