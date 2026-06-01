/** True when UI should use fixtures (dev flag or ?preview=1 in hash query). */
export function isPreviewForced(): boolean {
  if (import.meta.env.VITE_SITES_PREVIEW === '1') return true
  const hash = window.location.hash
  const q = hash.includes('?') ? hash.split('?')[1] : ''
  return new URLSearchParams(q).get('preview') === '1'
}

export function previewQuerySuffix(): string {
  return isPreviewForced() ? '?preview=1' : ''
}
