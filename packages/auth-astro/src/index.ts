import type { AuthClient, AuthUser } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';

export interface AstroAuth {
  user: AuthUser | null;
  session: any;
  token: string | null;
}

export function createBigBaseAuth(options: { baseURL: string; cookieSecret?: string }) {
  const client = createAuthClient({ baseURL: options.baseURL });

  return async (context: any, next: () => Promise<Response>) => {
    const cookieHeader = context.request?.headers?.get('cookie') ?? '';
    const token = cookieHeader.match(/token=([^;]+)/)?.[1] ?? null;

    if (token) {
      try {
        const session = await client.getSession();
        context.locals.auth = session;
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

export function requireAuth(astro: any): AstroAuth {
  const auth = getSession(astro);
  if (!auth) {
    return astro.redirect('/login');
  }
  return auth;
}

export { createAuthClient };
