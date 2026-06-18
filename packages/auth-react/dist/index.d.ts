import type { AuthClient, AuthUser, AuthSession } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';
export declare function useSession(client: AuthClient): {
    data: AuthSession | null;
    isPending: boolean;
    error: Error | null;
};
export declare function useAuth(options: {
    baseURL: string;
}): {
    client: AuthClient;
    user: AuthUser | null;
    session: AuthSession | null;
    isPending: boolean;
    error: Error | null;
    signIn: (email: string, password: string) => Promise<AuthSession>;
    signUp: (email: string, password: string) => Promise<AuthSession>;
    signOut: () => Promise<void>;
};
export { createAuthClient };
export type { AuthClient, AuthUser, AuthSession };
