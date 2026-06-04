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

  it('renders sticky app footer with prototype links', async () => {
    renderLayout()
    await screen.findByTestId('app-footer')
    expect(screen.getByText(/© 2026 BigBase/)).toBeInTheDocument()
    expect(screen.getByText(/MIT License/)).toBeInTheDocument()
    expect(screen.getByText('GitHub')).toBeInTheDocument()
    expect(screen.getByText('Changelog')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'BigPowers' })).toHaveAttribute('href', 'https://github.com/danielvm-git/bigpowers')
    expect(screen.getByRole('link', { name: 'danielvm-git' })).toHaveAttribute('href', 'https://github.com/danielvm-git')
    expect(screen.getByText('v2.0.0')).toBeInTheDocument()
  })

  it('uses scroll shell on layout body and content', async () => {
    const { container } = renderLayout()
    await screen.findByText('Home')
    const body = container.querySelector('.layout-body')
    const content = container.querySelector('.content')
    const footer = container.querySelector('.app-footer')
    expect(body).toBeTruthy()
    expect(content).toBeTruthy()
    expect(footer).toBeTruthy()
    expect(getComputedStyle(content!).overflowY).toBe('auto')
    expect(getComputedStyle(footer!).flexShrink).toBe('0')
  })
})
