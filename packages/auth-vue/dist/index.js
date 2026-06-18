import { createAuthClient } from '@bigbase/auth';
import { ref, inject } from 'vue';
const AUTH_KEY = Symbol('bigbase-auth');
export function createBigBaseAuth(options) {
    const client = createAuthClient({ baseURL: options.baseURL });
    return {
        install(app) {
            const user = ref(null);
            const session = ref(null);
            const isLoading = ref(true);
            const isAuthenticated = ref(false);
            client.getSession().then((s) => {
                user.value = s?.user ?? null;
                session.value = s;
                isAuthenticated.value = s !== null;
                isLoading.value = false;
            });
            client.onAuthStateChange((s) => {
                user.value = s?.user ?? null;
                session.value = s;
                isAuthenticated.value = s !== null;
                isLoading.value = false;
            });
            app.provide(AUTH_KEY, { user, session, isAuthenticated, isLoading, client });
        },
    };
}
export function useAuth() {
    const auth = inject(AUTH_KEY);
    if (!auth)
        throw new Error('useAuth() must be used within a BigBase auth plugin');
    return auth;
}
export function useSession() {
    const { client } = useAuth();
    return { client };
}
export { createAuthClient };
