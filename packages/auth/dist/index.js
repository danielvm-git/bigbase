function createAuthClient(options) {
    const { baseURL } = options;
    let storedToken = null;
    const listeners = new Set();
    function notify(session) {
        for (const cb of listeners)
            cb(session);
    }
    async function request(path, init) {
        const headers = {
            'Content-Type': 'application/json',
            ...init?.headers,
        };
        if (storedToken) {
            headers['Authorization'] = `Bearer ${storedToken}`;
        }
        const res = await fetch(`${baseURL}${path}`, { ...init, headers });
        if (!res.ok) {
            const body = await res.json().catch(() => ({}));
            throw { status: res.status, code: body.error || 'unknown', message: body.error || res.statusText };
        }
        return res.json();
    }
    async function getSession() {
        try {
            const data = await request('/api/auth/me');
            return { user: { id: data.id, email: data.email, role: data.role }, token: storedToken || '' };
        }
        catch {
            return null;
        }
    }
    return {
        signIn: {
            async email({ email, password }) {
                const data = await request('/api/auth/login', {
                    method: 'POST',
                    body: JSON.stringify({ email, password }),
                });
                storedToken = data.token;
                const session = { user: data.user, token: data.token };
                notify(session);
                return session;
            },
            social({ provider, redirectTo }) {
                const url = `${baseURL}/api/auth/oauth/${provider}${redirectTo ? `?redirect=${encodeURIComponent(redirectTo)}` : ''}`;
                window.location.href = url;
            },
            otp: {
                async send({ email }) {
                    return request('/api/auth/otp/send', { method: 'POST', body: JSON.stringify({ email }) });
                },
                async verify({ email, code }) {
                    const data = await request('/api/auth/otp/verify', {
                        method: 'POST',
                        body: JSON.stringify({ email, code }),
                    });
                    storedToken = data.token;
                    const session = { user: await getSession().then(s => s?.user), token: data.token };
                    notify(session);
                    return session;
                },
            },
            magicLink: {
                async send({ email, redirectTo }) {
                    return request('/api/auth/magic-link/send', { method: 'POST', body: JSON.stringify({ email, redirect_to: redirectTo }) });
                },
            },
            phoneOTP: {
                async send({ phone }) {
                    return request('/api/auth/phone/send', { method: 'POST', body: JSON.stringify({ phone }) });
                },
                async verify({ phone, code }) {
                    const data = await request('/api/auth/phone/verify', {
                        method: 'POST',
                        body: JSON.stringify({ phone, code }),
                    });
                    storedToken = data.token;
                    const session = { user: await getSession().then(s => s?.user), token: data.token };
                    notify(session);
                    return session;
                },
            },
        },
        signUp: {
            async email({ email, password, name }) {
                const data = await request('/api/auth/register', {
                    method: 'POST',
                    body: JSON.stringify({ email, password, name }),
                });
                storedToken = data.token;
                const session = { user: data.user, token: data.token };
                notify(session);
                return session;
            },
        },
        async signOut() {
            try {
                await fetch(`${baseURL}/api/auth/logout`, { method: 'POST' });
            }
            catch { /* ok */ }
            storedToken = null;
            notify(null);
        },
        getSession,
        async getJWT() {
            return storedToken || (await getSession())?.token || null;
        },
        onAuthStateChange(callback) {
            listeners.add(callback);
            return () => listeners.delete(callback);
        },
    };
}
export { createAuthClient };
