// Stub auth hooks for the Settings page.
// In a real build these would wrap /api/auth/* endpoints; for now they
// return a single hard-coded user so the page renders something useful.

export interface AuthUser {
  id: string
  email: string
  name: string
}

const CURRENT_USER: AuthUser = {
  id: 'u1',
  email: 'owner@example.com',
  name: 'Owner',
}

export function useCurrentUser(): {
  data: AuthUser | null
  isLoading: boolean
} {
  return { data: CURRENT_USER, isLoading: false }
}
