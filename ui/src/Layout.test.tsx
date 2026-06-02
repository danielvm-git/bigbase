import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from './context/ThemeContext'
import Layout from './Layout'

function renderLayout() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<div>Home</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('Layout IA shell', () => {
  beforeEach(() => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : String(input)
      if (url.includes('/api/auth/me')) {
        return Promise.resolve(new Response(JSON.stringify({ id: 1, email: 'admin@example.com' }), { status: 200 }))
      }
      if (url.includes('/api/version')) {
        return Promise.resolve(new Response(JSON.stringify({ version: '2.0.0' }), { status: 200 }))
      }
      return Promise.reject(new Error(`unmocked: ${url}`))
    })
  })

  it('renders prototype sidebar groups', async () => {
    renderLayout()
    expect(await screen.findByText('Build')).toBeInTheDocument()
    expect(screen.getByText('Engage')).toBeInTheDocument()
    expect(screen.getByText('Auth')).toBeInTheDocument()
    expect(screen.getByText('Sites')).toBeInTheDocument()
    expect(screen.getByText('Settings')).toBeInTheDocument()
    expect(screen.getByText('Appearance')).toBeInTheDocument()
  })
})
