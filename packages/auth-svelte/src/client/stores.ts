import type { AuthClient, AuthUser } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';
import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

export interface SvelteAuthStores {
  user: ReturnType<typeof writable<AuthUser | null>>;
  session: ReturnType<typeof writable<any>>;
  isAuthenticated: ReturnType<typeof derived<any, boolean>>;
  isLoading: ReturnType<typeof writable<boolean>>;
}

export function createAuthStores(client: AuthClient): SvelteAuthStores {
  const user = writable<AuthUser | null>(null);
  const session = writable<any>(null);
  const isLoading = writable(true);
  const isAuthenticated = derived(user, ($user) => $user !== null);

  if (browser) {
    client.getSession().then((s) => {
      user.set(s?.user ?? null);
      session.set(s);
      isLoading.set(false);
    });
    client.onAuthStateChange((s) => {
      user.set(s?.user ?? null);
      session.set(s);
      isLoading.set(false);
    });
  }

  return { user, session, isAuthenticated, isLoading };
}

export { createAuthClient };
export type { AuthClient };
