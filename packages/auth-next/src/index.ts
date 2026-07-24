import { NextResponse, type NextRequest } from 'next/server';
import { createAuthClient } from '@bigbase/auth';

export interface NextAuthConfig {
  baseURL: string;
  cookieSecret: string;
  loginUrl?: string;
}

export function bigBaseAuthMiddleware(config: NextAuthConfig) {
  const { baseURL, loginUrl = '/login' } = config;
  const client = createAuthClient({ baseURL });
  const skipRoutes = ['/api/auth', '/auth/', '/_next'];

  return async (request: NextRequest) => {
    const pathname = request.nextUrl.pathname;
    if (skipRoutes.some((r) => pathname.startsWith(r))) {
      return NextResponse.next();
    }

    const token = request.cookies.get('token')?.value;
    if (token) {
      try {
        client.setToken(token);
        const session = await client.getSession();
        if (session) {
          return NextResponse.next();
        }
      } catch {
        // getSession threw — treat as unauthenticated
      }
      request.cookies.delete('token');
    }

    return NextResponse.redirect(new URL(loginUrl, request.url));
  };
}

export function authApiHandler(config: { baseURL: string; cookieSecret: string }) {
  const { baseURL, cookieSecret } = config;

  const handler = async (request: Request, { params }: { params: { path: string[] } }) => {
    const path = params.path.join('/');
    const url = `${baseURL}/api/auth/${path}`;
    const headers = new Headers(request.headers);
    headers.set('x-forwarded-host', request.headers.get('host') || '');

    const res = await fetch(url, {
      method: request.method,
      headers,
      body: request.body,
    });

    const responseHeaders = new Headers(res.headers);
    const setCookie = res.headers.get('set-cookie');
    if (setCookie) {
      responseHeaders.set('set-cookie', setCookie);
    }

    return new Response(res.body, {
      status: res.status,
      headers: responseHeaders,
    });
  };

  return { GET: handler, POST: handler, PUT: handler, DELETE: handler, PATCH: handler };
}
