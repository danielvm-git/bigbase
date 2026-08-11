import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MessagingPage from './MessagingPage'

function mockFetchOk(data: unknown) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(data),
  } as Response)
}

function renderPage() {
  return render(
    <MemoryRouter>
      <MessagingPage />
    </MemoryRouter>,
  )
}

describe('MessagingPage send confirmation', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('opens a confirm dialog with an action summary before sending an email', async () => {
    mockFetchOk({ data: [] })

    renderPage()
    fireEvent.click(screen.getByText('Send test'))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Send Email' })).toBeInTheDocument()
    })

    fireEvent.change(screen.getByLabelText('To'), { target: { value: 'ops@example.com' } })
    fireEvent.change(screen.getByLabelText('Body'), { target: { value: 'Deploy finished' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send Email' }))

    // No send request yet — only the initial history fetch has happened
    expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledTimes(1)

    const dialog = await screen.findByRole('dialog', { name: 'Send message' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    // The dialog summarises the pending action
    expect(screen.getByText(/ops@example\.com/)).toBeInTheDocument()
    expect(screen.getByText(/delivered immediately and cannot be recalled/)).toBeInTheDocument()

    // Cancelling keeps the form as-is and sends nothing
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledTimes(1)
  })

  it('sends the email only after confirming the dialog', async () => {
    mockFetchOk({ data: [] })

    renderPage()
    fireEvent.click(screen.getByText('Send test'))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Send Email' })).toBeInTheDocument()
    })

    fireEvent.change(screen.getByLabelText('To'), { target: { value: 'ops@example.com' } })
    fireEvent.change(screen.getByLabelText('Body'), { target: { value: 'Deploy finished' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send Email' }))

    await screen.findByRole('dialog', { name: 'Send message' })
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => {
      const post = vi.mocked(globalThis.fetch).mock.calls.find(
        ([input]) => typeof input === 'string' && input.includes('/api/messaging/email'),
      )
      expect(post).toBeTruthy()
    })
    const post = vi.mocked(globalThis.fetch).mock.calls.find(
      ([input]) => typeof input === 'string' && input.includes('/api/messaging/email'),
    )!
    expect(post[1]?.method).toBe('POST')
    expect(JSON.parse(String(post[1]?.body))).toEqual({
      to: 'ops@example.com',
      subject: '',
      body: 'Deploy finished',
    })
  })

  it('confirms SMS and push sends with channel-specific summaries', async () => {
    mockFetchOk({ data: [] })

    renderPage()
    fireEvent.click(screen.getByText('Send test'))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Send Email' })).toBeInTheDocument()
    })

    // SMS
    fireEvent.click(screen.getByText('SMS'))
    fireEvent.change(screen.getByLabelText('To'), { target: { value: '+15551234567' } })
    fireEvent.change(screen.getByLabelText('Message'), { target: { value: 'Outage notice' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send SMS' }))

    const smsDialog = await screen.findByRole('dialog', { name: 'Send message' })
    expect(smsDialog).toHaveTextContent('+15551234567')
    expect(smsDialog).toHaveTextContent('Outage notice')
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => {
      const post = vi.mocked(globalThis.fetch).mock.calls.find(
        ([input]) => typeof input === 'string' && input.includes('/api/messaging/sms'),
      )
      expect(post).toBeTruthy()
    })

    // Push
    fireEvent.click(screen.getByText('PUSH'))
    fireEvent.change(screen.getByLabelText('Device token'), { target: { value: 'tok-123' } })
    fireEvent.change(screen.getByLabelText('Body'), { target: { value: 'New build live' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send Push' }))

    const pushDialog = await screen.findByRole('dialog', { name: 'Send message' })
    expect(pushDialog).toHaveTextContent('tok-123')
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => {
      const post = vi.mocked(globalThis.fetch).mock.calls.find(
        ([input]) => typeof input === 'string' && input.includes('/api/messaging/push'),
      )
      expect(post).toBeTruthy()
    })
  })

  it('wires hints on ambiguous channel fields via aria-describedby', async () => {
    mockFetchOk({ data: [] })

    renderPage()
    fireEvent.click(screen.getByText('Send test'))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Send Email' })).toBeInTheDocument()
    })

    expect(screen.getByText('Recipient email address, e.g. name@example.com')).toBeInTheDocument()
    expect((screen.getByLabelText('To') as HTMLInputElement).getAttribute('aria-describedby')).toBe('emailTo-hint')
    expect(document.getElementById('emailTo-hint')).toHaveTextContent('Recipient email address')

    fireEvent.click(screen.getByText('SMS'))
    expect(screen.getByText('Recipient phone number in E.164 format, e.g. +15551234567')).toBeInTheDocument()
    expect((screen.getByLabelText('To') as HTMLInputElement).getAttribute('aria-describedby')).toBe('smsTo-hint')

    fireEvent.click(screen.getByText('PUSH'))
    expect(screen.getByText('Push token registered by the device (APNs or FCM)')).toBeInTheDocument()
    expect((screen.getByLabelText('Device token') as HTMLInputElement).getAttribute('aria-describedby')).toBe('pushToken-hint')
  })
})
