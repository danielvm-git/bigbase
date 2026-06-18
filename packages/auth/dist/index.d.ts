export interface AuthClientOptions {
    /** Base URL of the BigBase server (e.g. 'http://localhost:8080') */
    baseURL: string;
    /** Enable anonymous token fallback when no user session exists */
    allowAnonymous?: boolean;
}
export interface AuthUser {
    id: number;
    email: string;
    role: string;
    name?: string;
    avatar_url?: string;
}
export interface AuthSession {
    user: AuthUser;
    token: string;
}
export interface AuthError {
    status: number;
    code: string;
    message: string;
}
export type AuthStateCallback = (session: AuthSession | null) => void;
export interface AuthClient {
    signIn: {
        email(params: {
            email: string;
            password: string;
        }): Promise<AuthSession>;
        social(params: {
            provider: 'google';
            redirectTo?: string;
        }): void;
        otp: {
            send(params: {
                email: string;
            }): Promise<{
                ok: boolean;
            }>;
            verify(params: {
                email: string;
                code: string;
            }): Promise<AuthSession>;
        };
        magicLink: {
            send(params: {
                email: string;
                redirectTo?: string;
            }): Promise<{
                ok: boolean;
            }>;
        };
        phoneOTP: {
            send(params: {
                phone: string;
            }): Promise<{
                ok: boolean;
            }>;
            verify(params: {
                phone: string;
                code: string;
            }): Promise<AuthSession>;
        };
    };
    signUp: {
        email(params: {
            email: string;
            password: string;
            name?: string;
        }): Promise<AuthSession>;
    };
    signOut(): Promise<void>;
    getSession(): Promise<AuthSession | null>;
    getJWT(): Promise<string | null>;
    onAuthStateChange(callback: AuthStateCallback): () => void;
}
declare function createAuthClient(options: AuthClientOptions): AuthClient;
export { createAuthClient };
