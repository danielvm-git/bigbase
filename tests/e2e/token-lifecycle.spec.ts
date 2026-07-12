// story: e78s01
// story: e76s01
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// Module-scoped auth state for scaffold tests.
const email = `e2e-token-${Date.now()}@test.com`;
const password = 'TestPass123!';
let authToken: string;
let refreshToken: string;

// Decode a JWT payload segment (base64url → JSON).
function decodeJWTPayload(token: string): Record<string, unknown> {
  const parts = token.split('.');
  expect(parts).toHaveLength(3);
  const payload = parts[1];
  // JWT uses base64url (no padding, - instead of +, _ instead of /)
  const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
  const decoded = atob(base64);
  return JSON.parse(decoded);
}

test.beforeAll(async ({ request }) => {
  // Register a fresh user for the scaffold tests.
  const regRes = await request.post('/api/auth/register', {
    data: { email, password },
  });
  expect(regRes.status()).toBe(201);
  const regBody = await regRes.json();

  authToken = regBody.token;
  refreshToken = regBody.refresh_token;

  expect(typeof authToken).toBe('string');
  expect(typeof refreshToken).toBe('string');
  // Verify refresh token is 64 hex chars (32 bytes).
  expect(refreshToken).toMatch(/^[0-9a-f]{64}$/);
  // Verify JWT is 3-segment.
  expect(authToken.split('.')).toHaveLength(3);
});

test.describe('Token Lifecycle E2E', () => {

  test('full session lifecycle (P0)', async ({ request }) => {
    // Use a dedicated user for this test to avoid cross-contamination.
    const ts = Date.now();
    const testEmail = `e2e-session-${ts}-${test.info().workerIndex}@test.com`;

    // 1. Register a new user.
    const regRes = await request.post('/api/auth/register', {
      data: { email: testEmail, password },
    });
    expect(regRes.status()).toBe(201);
    const regBody = await regRes.json();
    const jwt1 = regBody.token;
    const rt1 = regBody.refresh_token;
    expect(jwt1.split('.')).toHaveLength(3);
    expect(rt1).toMatch(/^[0-9a-f]{64}$/);

    // 2. Login with same credentials — get a new token pair.
    const loginRes = await request.post('/api/auth/login', {
      data: { email: testEmail, password },
    });
    expect(loginRes.status()).toBe(200);
    const loginBody = await loginRes.json();
    const jwt2 = loginBody.token;
    const rt2 = loginBody.refresh_token;
    expect(jwt2.split('.')).toHaveLength(3);
    expect(rt2).toMatch(/^[0-9a-f]{64}$/);
    // Refresh tokens are random — login must produce a different one.
    // JWT may be identical if created in the same second (iat has second precision).
    expect(rt2).not.toBe(rt1);

    // 3. Refresh with the latest refresh token.
    const refreshRes = await request.post('/api/auth/refresh', {
      data: { refresh_token: rt2 },
    });
    expect(refreshRes.status()).toBe(200);
    const refreshBody = await refreshRes.json();
    const jwt3 = refreshBody.token;
    const rt3 = refreshBody.refresh_token;
    expect(jwt3.split('.')).toHaveLength(3);
    expect(rt3).toMatch(/^[0-9a-f]{64}$/);
    // Refresh token must rotate; JWT may be identical within the same second.
    expect(rt3).not.toBe(rt2);

    // 4. Rotation: reuse the OLD refresh token (rt2) — must be rejected with 401.
    const replayRes = await request.post('/api/auth/refresh', {
      data: { refresh_token: rt2 },
    });
    expect(replayRes.status()).toBe(401);

    // 5. Logout-all with the current JWT.
    const logoutAllRes = await request.post('/api/auth/logout-all', {
      headers: { Authorization: `Bearer ${jwt3}` },
    });
    expect(logoutAllRes.status()).toBe(200);

    // 6. After logout-all, the current refresh token (rt3) is invalidated.
    const postLogoutRefreshRes = await request.post('/api/auth/refresh', {
      data: { refresh_token: rt3 },
    });
    expect(postLogoutRefreshRes.status()).toBe(401);

    // Note: Stateless JWT design — the JWT access token (jwt3) remains valid
    // until its 30s expiry. The server has no token blacklist.
    // Only refresh tokens are invalidated by logout-all.
  });

  test('org API key lifecycle (P0)', async ({ request }) => {
    const ts = Date.now();
    const testEmail = `e2e-orgkey-${ts}-${test.info().workerIndex}@test.com`;

    // Register a user for org management.
    const regRes = await request.post('/api/auth/register', {
      data: { email: testEmail, password },
    });
    expect(regRes.status()).toBe(201);
    const jwt = (await regRes.json()).token;

    // 1. Create an org.
    const slug = `e2e-orgkey-${ts}`;
    const orgRes = await request.post('/api/orgs', {
      headers: { Authorization: `Bearer ${jwt}`, 'Content-Type': 'application/json' },
      data: { name: slug, slug },
    });
    expect(orgRes.status()).toBe(201);
    const orgBody = await orgRes.json();
    const orgId = (orgBody.data as { id: number }).id;

    // 2. Generate a bb_* API key.
    const keyRes = await request.post(`/api/orgs/${orgId}/api-keys`, {
      headers: { Authorization: `Bearer ${jwt}`, 'Content-Type': 'application/json' },
      data: { name: 'e2e-test-key' },
    });
    expect(keyRes.status()).toBe(201);
    const keyBody = await keyRes.json();
    const apiKey = (keyBody.data as { key: string }).key;
    const keyId = (keyBody.data as { id: number }).id;

    // Assert key format: bb_ prefix + 64 hex chars = 67 chars total.
    expect(apiKey).toMatch(/^bb_[0-9a-f]{64}$/);
    expect(apiKey).toHaveLength(67);

    // 3. Authenticate with the API key on a protected endpoint.
    const healthRes = await request.get('/api/sites', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(healthRes.status()).toBe(200);

    // 4. Revoke the key.
    const revokeRes = await request.delete(`/api/orgs/${orgId}/api-keys/${keyId}`, {
      headers: { Authorization: `Bearer ${jwt}` },
    });
    expect(revokeRes.status()).toBe(200);

    // 5. Authenticate again with the same key — must be rejected.
    const postRevokeRes = await request.get('/api/sites', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(postRevokeRes.status()).toBe(401);
  });

  test('JWT lifetime enforcement (P1)', async ({ request }) => {
    const ts = Date.now();
    const testEmail = `e2e-jwt-${ts}-${test.info().workerIndex}@test.com`;

    // Register and login to get a JWT.
    const regRes = await request.post('/api/auth/register', {
      data: { email: testEmail, password },
    });
    expect(regRes.status()).toBe(201);
    const token: string = (await regRes.json()).token;

    // Decode the JWT payload.
    const payload = decodeJWTPayload(token);

    // Validate 3-segment format (already done in decode).
    expect(token.split('.')).toHaveLength(3);

    // Validate claims.
    expect(payload).toHaveProperty('exp');
    expect(payload).toHaveProperty('iat');
    const exp = payload.exp as number;
    const iat = payload.iat as number;

    // exp - iat should be approximately 30s (config window).
    const ttl = exp - iat;
    expect(ttl).toBeGreaterThanOrEqual(25);
    expect(ttl).toBeLessThanOrEqual(35);

    // iat should be roughly now (within a few seconds).
    const now = Math.floor(Date.now() / 1000);
    expect(Math.abs(iat - now)).toBeLessThanOrEqual(10);

    // Verify standard fields.
    expect(payload).toHaveProperty('user_id');
    expect(payload).toHaveProperty('email', testEmail);
    // Role depends on registration order — just verify it's a non-empty string.
    expect(payload).toHaveProperty('role');
    expect(typeof payload.role).toBe('string');
    expect((payload.role as string).length).toBeGreaterThan(0);
    expect(payload).toHaveProperty('org_id');
  });

  test('log redaction (P2)', async ({ request }) => {
    // Send a request with an invalid token and assert 401.
    const badToken = 'MySecretToken123';
    const res = await request.get('/api/sites', {
      headers: { Authorization: `Bearer ${badToken}` },
    });
    expect(res.status()).toBe(401);

    // Note: Server log verification is skipped because Playwright
    // does not expose the webServer process's stdout/stderr to test
    // runners. To verify CWE-532 compliance (no raw token in logs):
    // - Run the server separately: go run . serve --port 9999 --db /tmp/bigbase-e2e.db
    // - Send a request with a distinct token like "TEST-SECRET-TOKEN-VALUE"
    // - Check that this string does NOT appear in the server's stdout/stderr.
    // File a follow-up to add log capture infrastructure.
  });
});
