import { createAuthClient } from '@bigbase/auth';
import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';
export function createAuthStores(client) {
    const user = writable(null);
    const session = writable(null);
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
