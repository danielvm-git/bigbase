import type { AuthClient, AuthUser, AuthSession } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';

export interface AstroAuth {
  user: AuthUser | null;
  session: AuthSession | null;
  token: string | null;
}

export interface AstroAuthLocals {
  auth: AstroAuth | null;
}

export function createBigBaseAuth(options: { baseURL: string; cookieSecret?: string }) {
  const client = createAuthClient({ baseURL: options.baseURL });

  return async (context: any, next: () => Promise<Response>) => {
    const cookieHeader = context.request?.headers?.get('cookie') ?? '';
    const token = cookieHeader.match(/token=([^;]+)/)?.[1] ?? null;

    if (token) {
      try {
        client.setToken(token);
        const session = await client.getSession();
        context.locals.auth = session ? {
          user: session.user,
          session: session,
          token: session.token
        } : null;
      } catch {
        context.locals.auth = null;
      }
    } else {
      context.locals.auth = null;
    }

    return next();
  };
}

export function getSession(astro: any): AstroAuth | null {
  return astro.locals?.auth ?? null;
}

export function requireAuth(astro: any): AstroAuth | Response {
  const auth = getSession(astro);
  if (!auth) {
    return astro.redirect('/login');
  }
  return auth;
}

export { createAuthClient };
