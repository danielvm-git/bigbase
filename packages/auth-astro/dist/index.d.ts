import type { AuthUser } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';
export interface AstroAuth {
    user: AuthUser | null;
    session: any;
    token: string | null;
}
export declare function createBigBaseAuth(options: {
    baseURL: string;
    cookieSecret?: string;
}): (context: any, next: () => Promise<Response>) => Promise<Response>;
export declare function getSession(astro: any): AstroAuth | null;
export declare function requireAuth(astro: any): AstroAuth;
export { createAuthClient };
