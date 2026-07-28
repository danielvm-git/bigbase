import '@testing-library/jest-dom'
import './index.css'

// Ensure localStorage exists even under Node 26's experimental built-in (BUG-000003).
// jsdom normally provides it, but Node 26's gated built-in can shadow it as undefined.
if (typeof globalThis.localStorage === 'undefined') {
  const store = new Map<string, string>()
  const localStorageStub: Storage = {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => { store.set(k, String(v)) },
    removeItem: (k: string) => { store.delete(k) },
    clear: () => { store.clear() },
    key: (i: number) => Array.from(store.keys())[i] ?? null,
    get length() { return store.size },
  }
  Object.defineProperty(globalThis, 'localStorage', {
    value: localStorageStub,
    configurable: true,
    writable: true,
  })
}
