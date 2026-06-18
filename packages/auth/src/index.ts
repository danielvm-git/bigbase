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
    email(params: { email: string; password: string }): Promise<AuthSession>;
    social(params: { provider: 'google'; redirectTo?: string }): void;
    otp: {
      send(params: { email: string }): Promise<{ ok: boolean }>;
      verify(params: { email: string; code: string }): Promise<AuthSession>;
    };
    magicLink: {
      send(params: { email: string; redirectTo?: string }): Promise<{ ok: boolean }>;
    };
    phoneOTP: {
      send(params: { phone: string }): Promise<{ ok: boolean }>;
      verify(params: { phone: string; code: string }): Promise<AuthSession>;
    };
  };
  signUp: {
    email(params: { email: string; password: string; name?: string }): Promise<AuthSession>;
  };
  signOut(): Promise<void>;
  getSession(): Promise<AuthSession | null>;
  getJWT(): Promise<string | null>;
  onAuthStateChange(callback: AuthStateCallback): () => void;
}

function createAuthClient(options: AuthClientOptions): AuthClient {
  const { baseURL } = options;
  let storedToken: string | null = null;
  const listeners = new Set<AuthStateCallback>();

  function notify(session: AuthSession | null) {
    for (const cb of listeners) cb(session);
  }

  async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(init?.headers as Record<string, string>),
    };
    if (storedToken) {
      headers['Authorization'] = `Bearer ${storedToken}`;
    }
    const res = await fetch(`${baseURL}${path}`, { ...init, headers });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw { status: res.status, code: body.error || 'unknown', message: body.error || res.statusText } as AuthError;
    }
    return res.json() as T;
  }

  async function getSession(): Promise<AuthSession | null> {
    try {
      const data = await request<{ id: number; email: string; role: string }>('/api/auth/me');
      return { user: { id: data.id, email: data.email, role: data.role }, token: storedToken || '' };
    } catch {
      return null;
    }
  }

  return {
    signIn: {
      async email({ email, password }) {
        const data = await request<{ token: string; user: { id: number; email: string; role: string } }>('/api/auth/login', {
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
          const data = await request<{ token: string }>('/api/auth/otp/verify', {
            method: 'POST',
            body: JSON.stringify({ email, code }),
          });
          storedToken = data.token;
          const session = { user: await getSession().then(s => s?.user!), token: data.token };
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
          const data = await request<{ token: string }>('/api/auth/phone/verify', {
            method: 'POST',
            body: JSON.stringify({ phone, code }),
          });
          storedToken = data.token;
          const session = { user: await getSession().then(s => s?.user!), token: data.token };
          notify(session);
          return session;
        },
      },
    },
    signUp: {
      async email({ email, password, name }) {
        const data = await request<{ token: string; user: { id: number; email: string; role: string } }>('/api/auth/register', {
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
      try { await fetch(`${baseURL}/api/auth/logout`, { method: 'POST' }); } catch { /* ok */ }
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
