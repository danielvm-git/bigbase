import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import StoragePage from './StoragePage'

const mockFiles = [
  { id: '1', name: 'photo.png', size: 204800, mime_type: 'image/png', created_at: '2026-06-01T10:00:00Z' },
  { id: '2', name: 'doc.pdf', size: 1024, mime_type: 'application/pdf', created_at: '2026-06-01T11:00:00Z' },
  { id: '3', name: 'readme.md', size: 512, mime_type: 'text/markdown', created_at: '2026-06-01T12:00:00Z' },
]

function mockFetchOk(data: unknown) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(data),
  } as Response)
}

describe('StoragePage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the page header', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    } as Response)

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Storage')).toBeInTheDocument()
    })
  })

  it('shows empty state when no files exist', async () => {
    mockFetchOk({ data: [] })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.queryByRole('row', { name: /photo/ })).not.toBeInTheDocument()
      expect(screen.queryByText(/photo/)).not.toBeInTheDocument()
    })
  })

  it('renders file list in default list mode', async () => {
    mockFetchOk({ data: mockFiles })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('photo.png')).toBeInTheDocument()
      expect(screen.getByText('doc.pdf')).toBeInTheDocument()
      expect(screen.getByText('readme.md')).toBeInTheDocument()
    })
  })

  it('displays file sizes formatted correctly', async () => {
    mockFetchOk({ data: mockFiles })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('200.0 KB')).toBeInTheDocument()
      expect(screen.getByText('1.0 KB')).toBeInTheDocument()
      expect(screen.getByText('512 B')).toBeInTheDocument()
    })
  })

  it('toggles to grid view and back', async () => {
    mockFetchOk({ data: mockFiles })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('photo.png')).toBeInTheDocument()
    })

    const gridButton = screen.getByText('Grid')
    const listButton = screen.getByText('List')

    fireEvent.click(gridButton)

    // After clicking Grid, the Grid button should have primary variant class, List secondary
    expect(gridButton.className).toContain('primary')
    expect(listButton.className).toContain('secondary')
  })

  it('shows error message on fetch failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('network error')).toBeInTheDocument()
    })
  })

  it('shows upload button', async () => {
    mockFetchOk({ data: [] })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Upload')).toBeInTheDocument()
    })
  })

  it('renders file input for uploads', async () => {
    mockFetchOk({ data: [] })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      const fileInput = document.querySelector('input[type="file"]')
      expect(fileInput).toBeInTheDocument()
    })
  })

  it('opens preview modal for image files', async () => {
    mockFetchOk({ data: mockFiles })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('photo.png')).toBeInTheDocument()
    })

    // Find the preview trigger - click on the image file name link
    const photoLink = screen.getByText('photo.png')
    fireEvent.click(photoLink)

    await waitFor(() => {
      // Preview must be a real modal dialog with the file name and a close button
      const dialog = screen.getByRole('dialog', { name: /photo\.png/ })
      expect(dialog).toHaveAttribute('aria-modal', 'true')
      expect(screen.getByRole('button', { name: 'Close dialog' })).toBeInTheDocument()
    })
  })

  it('closes preview modal when Close is clicked', async () => {
    mockFetchOk({ data: mockFiles })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('photo.png')).toBeInTheDocument()
    })

    const photoLink = screen.getByText('photo.png')
    fireEvent.click(photoLink)

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Close dialog' }))

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  it('deletes a file when confirmed', async () => {
    mockFetchOk({ data: mockFiles })

    vi.spyOn(window, 'confirm').mockReturnValueOnce(true)

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('photo.png')).toBeInTheDocument()
    })

    // Mock the DELETE call to succeed, then re-fetch with one fewer file
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    } as Response)
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: mockFiles.slice(1) }),
    } as Response)

    // Find and click the delete button for photo.png
    const deleteButtons = screen.getAllByText('Delete')
    // Delete the first file
    fireEvent.click(deleteButtons[0])

    await waitFor(() => {
      // After delete, the file should no longer appear
      expect(screen.queryByText('photo.png')).not.toBeInTheDocument()
      expect(screen.getByText('doc.pdf')).toBeInTheDocument()
      expect(screen.getByText('readme.md')).toBeInTheDocument()
    })
  })

  it('does not delete when confirmation is cancelled', async () => {
    mockFetchOk({ data: mockFiles })

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false)

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('photo.png')).toBeInTheDocument()
    })

    const deleteButtons = screen.getAllByText('Delete')
    fireEvent.click(deleteButtons[0])

    // File should still be present
    await waitFor(() => {
      expect(screen.getByText('photo.png')).toBeInTheDocument()
    })
  })

  it('shows Refresh button', async () => {
    mockFetchOk({ data: [] })

    render(<MemoryRouter><StoragePage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Refresh')).toBeInTheDocument()
    })
  })
})
