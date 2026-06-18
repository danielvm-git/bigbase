import { createAuthClient } from '@bigbase/auth';
export function bigBaseAuthHandle(options) {
    const { baseURL, loginUrl = '/login' } = options;
    const client = createAuthClient({ baseURL });
    return async ({ event, resolve }) => {
        const token = event.cookies.get('token') ?? null;
        if (token) {
            try {
                const session = await client.getSession();
                event.locals.auth = session;
            }
            catch {
                event.cookies.delete('token', { path: '/' });
                event.locals.auth = null;
            }
        }
        else {
            event.locals.auth = null;
        }
        // Protect routes (optional)
        const pathname = event.url.pathname;
        if (pathname.startsWith('/api/') || pathname.startsWith('/auth/')) {
            return resolve(event);
        }
        return resolve(event);
    };
}
export function getSession(event) {
    return event.locals.auth ?? null;
}
export async function requireAuth(event) {
    const auth = getSession(event);
    if (!auth) {
        throw new Response(null, { status: 307, headers: { location: '/login' } });
    }
    return auth;
}
