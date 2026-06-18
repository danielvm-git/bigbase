import type { AuthClient, AuthUser, AuthSession } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';
import { type Ref, type Plugin } from 'vue';
export interface VueAuthState {
    user: Ref<AuthUser | null>;
    session: Ref<AuthSession | null>;
    isAuthenticated: Ref<boolean>;
    isLoading: Ref<boolean>;
    client: AuthClient;
}
export declare function createBigBaseAuth(options: {
    baseURL: string;
}): Plugin;
export declare function useAuth(): VueAuthState;
export declare function useSession(): {
    client: AuthClient;
};
export { createAuthClient };
