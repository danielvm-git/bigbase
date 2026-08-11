import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useDocumentTitle } from './useDocumentTitle'

describe('useDocumentTitle', () => {
  beforeEach(() => {
    document.title = 'BigBase Admin'
  })

  it('sets document.title to the given title', () => {
    renderHook(() => useDocumentTitle('Users — BigBase Admin'))
    expect(document.title).toBe('Users — BigBase Admin')
  })

  it('restores the previous title on unmount', () => {
    const { unmount } = renderHook(() => useDocumentTitle('Dashboard — BigBase Admin'))
    expect(document.title).toBe('Dashboard — BigBase Admin')
    unmount()
    expect(document.title).toBe('BigBase Admin')
  })

  it('updates the title when the given title changes', () => {
    const { rerender } = renderHook(({ title }: { title: string }) => useDocumentTitle(title), {
      initialProps: { title: 'SQL Editor — BigBase Admin' },
    })
    expect(document.title).toBe('SQL Editor — BigBase Admin')
    rerender({ title: 'Sites — BigBase Admin' })
    expect(document.title).toBe('Sites — BigBase Admin')
  })
})
