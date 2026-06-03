// Stub workspace hooks for the Settings page.
// In a real build these would wrap /api/workspace/* and /api/users; for
// now they return a single hard-coded workspace and member list so the
// page renders something useful.

export interface Workspace {
  id: string
  name: string
}

export interface Member {
  id: string
  email: string
  role: 'owner' | 'admin' | 'member'
}

const WORKSPACE: Workspace = { id: 'w1', name: 'My Workspace' }
const MEMBERS: Member[] = [
  { id: 'm1', email: 'alice@example.com', role: 'admin' },
  { id: 'm2', email: 'bob@example.com', role: 'member' },
]

export function useWorkspace(): {
  data: Workspace | null
  isLoading: boolean
} {
  return { data: WORKSPACE, isLoading: false }
}

export function useMembers(): {
  data: Member[]
  isLoading: boolean
} {
  return { data: MEMBERS, isLoading: false }
}
