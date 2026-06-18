import type { AuthClient, AuthUser } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';
import { writable, derived } from 'svelte/store';
export interface SvelteAuthStores {
    user: ReturnType<typeof writable<AuthUser | null>>;
    session: ReturnType<typeof writable<any>>;
    isAuthenticated: ReturnType<typeof derived<any, boolean>>;
    isLoading: ReturnType<typeof writable<boolean>>;
}
export declare function createAuthStores(client: AuthClient): SvelteAuthStores;
export { createAuthClient };
export type { AuthClient };
