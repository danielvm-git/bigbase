import { NextResponse } from 'next/server';
import { createAuthClient } from '@bigbase/auth';
export function bigBaseAuthMiddleware(config) {
    const { baseURL, loginUrl = '/login' } = config;
    const client = createAuthClient({ baseURL });
    const skipRoutes = ['/api/auth', '/auth/', '/_next'];
    return async (request) => {
        const pathname = request.nextUrl.pathname;
        if (skipRoutes.some((r) => pathname.startsWith(r))) {
            return NextResponse.next();
        }
        const token = request.cookies.get('token')?.value;
        if (token) {
            try {
                await client.getSession();
                return NextResponse.next();
            }
            catch {
                request.cookies.delete('token');
            }
        }
        return NextResponse.redirect(new URL(loginUrl, request.url));
    };
}
export function authApiHandler(config) {
    const { baseURL, cookieSecret } = config;
    const handler = async (request, { params }) => {
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
