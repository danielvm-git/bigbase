import type { Handle, RequestEvent } from '@sveltejs/kit';
import { createAuthClient } from '@bigbase/auth';

export interface SvelteKitAuthOptions {
  baseURL: string;
  cookieSecret: string;
  loginUrl?: string;
}

export function bigBaseAuthHandle(options: SvelteKitAuthOptions): Handle {
  const { baseURL, loginUrl = '/login' } = options;
  const client = createAuthClient({ baseURL });

  return async ({ event, resolve }) => {
    const token = event.cookies.get('token') ?? null;
    if (token) {
      try {
        const session = await client.getSession();
        (event.locals as any).auth = session;
      } catch {
        event.cookies.delete('token', { path: '/' });
        (event.locals as any).auth = null;
      }
    } else {
      (event.locals as any).auth = null;
    }

    // Protect routes (optional)
    const pathname = event.url.pathname;
    if (pathname.startsWith('/api/') || pathname.startsWith('/auth/')) {
      return resolve(event);
    }

    return resolve(event);
  };
}

export function getSession(event: RequestEvent) {
  return (event.locals as any).auth ?? null;
}

export async function requireAuth(event: RequestEvent) {
  const auth = getSession(event);
  if (!auth) {
    throw new Response(null, { status: 307, headers: { location: '/login' } });
  }
  return auth;
}
