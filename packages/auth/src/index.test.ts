import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createAuthClient } from './index';
import { createBigBaseAuth } from '../../auth-astro/src/index';

// Mock fetch globally
const mockFetch = vi.fn();
(globalThis as any).fetch = mockFetch;

function jsonResponse(data: any, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
  } as Response);
}

beforeEach(() => {
  mockFetch.mockReset();
});

describe('createAuthClient', () => {
  it('signs in with email and password', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ token: 'jwt-123', user: { id: 1, email: 'test@test.com', role: 'user' } }));

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    const session = await client.signIn.email({ email: 'test@test.com', password: 'pass' });

    expect(session.token).toBe('jwt-123');
    expect(session.user.email).toBe('test@test.com');
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/auth/login',
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('signs up with email', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ token: 'jwt-456', user: { id: 2, email: 'new@test.com', role: 'user' } }));

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    const session = await client.signUp.email({ email: 'new@test.com', password: 'pass', name: 'New User' });

    expect(session.token).toBe('jwt-456');
  });

  it('sends OTP', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }));

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    const result = await client.signIn.otp.send({ email: 'test@test.com' });

    expect(result.ok).toBe(true);
  });

  it('sends magic link', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }));

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    const result = await client.signIn.magicLink.send({ email: 'test@test.com', redirectTo: '/dashboard' });

    expect(result.ok).toBe(true);
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.redirect_to).toBe('/dashboard');
  });

  it('signs out', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }));

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    await client.signOut();

    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/auth/logout',
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('gets session', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, email: 'test@test.com', role: 'user' }));

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    const session = await client.getSession();

    expect(session?.user.email).toBe('test@test.com');
  });

  it('returns null session on error', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, status: 401, json: () => Promise.resolve({ error: 'unauthorized' }) } as any);

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    const session = await client.getSession();

    expect(session).toBeNull();
  });

  it('fires onAuthStateChange on sign in', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ token: 'jwt', user: { id: 1, email: 'a@b.com', role: 'user' } }));
    mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }));

    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    const calls: any[] = [];
    client.onAuthStateChange((s) => calls.push(s));

    await client.signIn.email({ email: 'a@b.com', password: 'x' });
    expect(calls).toHaveLength(1);
    expect(calls[0]?.user.email).toBe('a@b.com');

    await client.signOut();
    expect(calls).toHaveLength(2);
    expect(calls[1]).toBeNull();
  });

  it('sets token manually using setToken', async () => {
    const client = createAuthClient({ baseURL: 'http://localhost:8080' });
    client.setToken('manual-jwt-123');
    const token = await client.getJWT();
    expect(token).toBe('manual-jwt-123');
    expect(mockFetch).not.toHaveBeenCalled();
  });
});

describe('createBigBaseAuth middleware', () => {
  it('extracts token from cookie, sets it on client, and populates locals.auth', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, email: 'user@test.com', role: 'user' }));

    const middleware = createBigBaseAuth({ baseURL: 'http://localhost:8080' });
    const context = {
      request: {
        headers: {
          get(name: string) {
            if (name === 'cookie') return 'token=jwt123';
            return null;
          }
        }
      },
      locals: {} as any
    };
    const next = vi.fn().mockResolvedValue(new Response());

    await middleware(context, next);

    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/auth/me',
      expect.objectContaining({
        headers: expect.objectContaining({
          'Authorization': 'Bearer jwt123'
        })
      })
    );
    expect(context.locals.auth).toEqual({
      user: { id: 1, email: 'user@test.com', role: 'user' },
      session: {
        user: { id: 1, email: 'user@test.com', role: 'user' },
        token: 'jwt123'
      },
      token: 'jwt123'
    });
    expect(next).toHaveBeenCalled();
  });

  it('sets locals.auth to null when cookie is missing', async () => {
    const middleware = createBigBaseAuth({ baseURL: 'http://localhost:8080' });
    const context = {
      request: {
        headers: {
          get() {
            return null;
          }
        }
      },
      locals: {} as any
    };
    const next = vi.fn().mockResolvedValue(new Response());

    await middleware(context, next);

    expect(mockFetch).not.toHaveBeenCalled();
    expect(context.locals.auth).toBeNull();
    expect(next).toHaveBeenCalled();
  });
});
