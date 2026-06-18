import type { AuthClient, AuthUser, AuthSession } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';
import { ref, inject, provide, type Ref, type App, type Plugin } from 'vue';

export interface VueAuthState {
  user: Ref<AuthUser | null>;
  session: Ref<AuthSession | null>;
  isAuthenticated: Ref<boolean>;
  isLoading: Ref<boolean>;
  client: AuthClient;
}

const AUTH_KEY = Symbol('bigbase-auth');

export function createBigBaseAuth(options: { baseURL: string }): Plugin {
  const client = createAuthClient({ baseURL: options.baseURL });

  return {
    install(app: App) {
      const user = ref<AuthUser | null>(null);
      const session = ref<AuthSession | null>(null);
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

export function useAuth(): VueAuthState {
  const auth = inject<VueAuthState>(AUTH_KEY);
  if (!auth) throw new Error('useAuth() must be used within a BigBase auth plugin');
  return auth;
}

export function useSession() {
  const { client } = useAuth();
  return { client };
}

export { createAuthClient };
