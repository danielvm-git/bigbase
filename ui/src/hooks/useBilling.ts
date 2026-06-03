// Stub billing hooks for the Settings page.
// In a real build these would wrap /api/billing; for now they return
// hard-coded plan and usage data so the page renders something useful.

export interface Billing {
  plan: 'free' | 'pro' | 'enterprise'
  renews: string
}

export interface Usage {
  functions: number
  storage_mb: number
  sites: number
}

const BILLING: Billing = { plan: 'pro', renews: '2026-12-31' }
const USAGE: Usage = { functions: 12, storage_mb: 480, sites: 4 }

export function useBilling(): {
  data: Billing | null
  isLoading: boolean
} {
  return { data: BILLING, isLoading: false }
}

export function useUsage(): {
  data: Usage | null
  isLoading: boolean
} {
  return { data: USAGE, isLoading: false }
}
