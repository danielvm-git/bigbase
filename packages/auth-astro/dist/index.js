import { createAuthClient } from '@bigbase/auth';
export function createBigBaseAuth(options) {
    const client = createAuthClient({ baseURL: options.baseURL });
    return async (context, next) => {
        const cookieHeader = context.request?.headers?.get('cookie') ?? '';
        const token = cookieHeader.match(/token=([^;]+)/)?.[1] ?? null;
        if (token) {
            try {
                const session = await client.getSession();
                context.locals.auth = session;
            }
            catch {
                context.locals.auth = null;
            }
        }
        else {
            context.locals.auth = null;
        }
        return next();
    };
}
export function getSession(astro) {
    return astro.locals?.auth ?? null;
}
export function requireAuth(astro) {
    const auth = getSession(astro);
    if (!auth) {
        return astro.redirect('/login');
    }
    return auth;
}
export { createAuthClient };
