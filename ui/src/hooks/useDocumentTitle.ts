import { useEffect } from 'react'

/**
 * Sets `document.title` for the current page and restores the previous
 * title when the component unmounts (WCAG 2.4.2 Page Titled).
 */
export function useDocumentTitle(title: string) {
  useEffect(() => {
    const previousTitle = document.title
    document.title = title
    return () => {
      document.title = previousTitle
    }
  }, [title])
}
