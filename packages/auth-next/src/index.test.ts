import { describe, it, expect, vi, beforeEach } from 'vitest';
import { bigBaseAuthMiddleware } from './index';

// Mock @bigbase/auth
const mockSetToken = vi.fn();
const mockGetSession = vi.fn();

vi.mock('@bigbase/auth', () => ({
  createAuthClient: () => ({
    setToken: mockSetToken,
    getSession: mockGetSession,
  }),
}));

// Minimal NextRequest mock
function makeRequest(pathname: string, cookieToken?: string) {
  const baseUrl = 'http://localhost:3000';
  const url = new URL(`${baseUrl}${pathname}`);
  return {
    url: `${baseUrl}${pathname}`,
    nextUrl: url,
    cookies: {
      get: (name: string) =>
        name === 'token' && cookieToken !== undefined
          ? { value: cookieToken }
          : undefined,
      delete: vi.fn(),
    },
  } as any;
}

// Minimal NextResponse mock — we only care about status/redirect behavior
vi.mock('next/server', () => ({
  NextResponse: {
    next: () => ({ type: 'next', status: 200 }),
    redirect: (url: URL) => ({ type: 'redirect', status: 307, url: url.toString() }),
  },
}));

describe('bigBaseAuthMiddleware', () => {
  const config = {
    baseURL: 'http://bigbase:8080',
    cookieSecret: 'secret',
    loginUrl: '/login',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('skips auth for /api/auth routes', async () => {
    const middleware = bigBaseAuthMiddleware(config);
    const res = await middleware(makeRequest('/api/auth/login'));
    expect(res.type).toBe('next');
  });

  it('skips auth for /_next routes', async () => {
    const middleware = bigBaseAuthMiddleware(config);
    const res = await middleware(makeRequest('/_next/static/chunk.js'));
    expect(res.type).toBe('next');
  });

  it('redirects to login when no token cookie', async () => {
    const middleware = bigBaseAuthMiddleware(config);
    const res = await middleware(makeRequest('/dashboard'));
    expect(res.type).toBe('redirect');
    expect(res.url).toContain('/login');
  });

  it('redirects to login when token cookie is empty string', async () => {
    const middleware = bigBaseAuthMiddleware(config);
    const res = await middleware(makeRequest('/dashboard', ''));
    // Empty string is falsy — should redirect
    expect(res.type).toBe('redirect');
    expect(res.url).toContain('/login');
  });

  it('calls setToken before getSession', async () => {
    mockGetSession.mockResolvedValueOnce({ user: { id: 1, email: 'a@b.com', role: 'admin' }, token: 'valid-jwt' });
    const middleware = bigBaseAuthMiddleware(config);
    await middleware(makeRequest('/dashboard', 'valid-jwt'));

    expect(mockSetToken).toHaveBeenCalledWith('valid-jwt');
    expect(mockGetSession).toHaveBeenCalledAfter(mockSetToken);
  });

  it('allows access when session is valid', async () => {
    mockGetSession.mockResolvedValueOnce({ user: { id: 1, email: 'a@b.com', role: 'admin' }, token: 'valid-jwt' });
    const middleware = bigBaseAuthMiddleware(config);
    const res = await middleware(makeRequest('/dashboard', 'valid-jwt'));
    expect(res.type).toBe('next');
  });

  it('redirects when getSession returns null (REGRESSION: BUG-138)', async () => {
    // This is the critical regression test — garbage token must NOT bypass auth.
    // Before the fix, getSession() returned null but the middleware still called
    // NextResponse.next() because it never checked the return value.
    mockGetSession.mockResolvedValueOnce(null);
    const middleware = bigBaseAuthMiddleware(config);
    const req = makeRequest('/dashboard', 'garbage-token');
    const res = await middleware(req);

    expect(mockSetToken).toHaveBeenCalledWith('garbage-token');
    expect(res.type).toBe('redirect');
    expect(res.url).toContain('/login');
    expect(req.cookies.delete).toHaveBeenCalledWith('token');
  });

  it('redirects when getSession throws', async () => {
    mockGetSession.mockRejectedValueOnce(new Error('network error'));
    const middleware = bigBaseAuthMiddleware(config);
    const req = makeRequest('/dashboard', 'some-token');
    const res = await middleware(req);

    expect(res.type).toBe('redirect');
    expect(res.url).toContain('/login');
    expect(req.cookies.delete).toHaveBeenCalledWith('token');
  });
});
