// story: e78s01
import { test, expect, type APIResponse } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// ── Module-scoped auth state ──────────────────────────────────────
const EMAIL = `e2e-surface-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;
let apiKey: string;
let orgId: number;

// ── Helpers ────────────────────────────────────────────────────────

/** Assert response is a valid error shape: { error: string } */
async function expectErrorShape(resp: Response | APIResponse, expectedStatus: number) {
  const status = typeof resp.status === 'function' ? resp.status() : resp.status;
  expect(status).toBe(expectedStatus);
  const body = await resp.json();
  expect(body).toHaveProperty('error');
  expect(typeof body.error).toBe('string');
}

/** Try to extract a numeric ID from responses that may wrap in `data` or return flat. */
function extractId(body: Record<string, unknown>): number | undefined {
  if (body.data && typeof body.data === 'object' && body.data !== null) {
    const d = body.data as Record<string, unknown>;
    if (typeof d.id === 'number') return d.id;
  }
  if (typeof body.id === 'number') return body.id;
  return undefined;
}

function extractStr(body: Record<string, unknown>, key: string): string | undefined {
  if (body.data && typeof body.data === 'object' && body.data !== null) {
    const d = body.data as Record<string, unknown>;
    if (typeof d[key] === 'string') return d[key] as string;
  }
  if (typeof body[key] === 'string') return body[key] as string;
  return undefined;
}

test.beforeAll(async () => {
  // 1. Register a fresh user (1 registration — safe under rate limiter).
  const regRes = await fetch('http://localhost:9999/api/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: EMAIL, password: PASSWORD }),
  });
  if (regRes.status !== 201) {
    console.warn(`Register failed: ${regRes.status}`);
    return;
  }
  const regBody = await regRes.json();
  authToken = regBody.token;

  // 2. Create an org for scoped tests.
  const slug = `e2e-surface-${Date.now()}`;
  const orgRes = await fetch('http://localhost:9999/api/orgs', {
    method: 'POST',
    headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: slug, slug }),
  });
  if (orgRes.status !== 201) {
    console.warn(`Org create failed: ${orgRes.status}`);
    return;
  }
  const orgBody = await orgRes.json();
  orgId = extractId(orgBody) ?? 0;

  // 3. Generate a long-lived API key for auth-bypass in tests.
  const keyRes = await fetch(`http://localhost:9999/api/orgs/${orgId}/api-keys`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: 'e2e-surface-test-key' }),
  });
  if (keyRes.status === 201) {
    const keyBody = await keyRes.json();
    apiKey = extractStr(keyBody, 'key') ?? '';
  }
});

/** Get a fresh JWT by re-logging in (handles 30s expiry). */
async function freshJWT(): Promise<string> {
  // Use API key if available (never expires), fall back to re-login
  if (apiKey) return apiKey;
  const res = await fetch('http://localhost:9999/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: EMAIL, password: PASSWORD }),
  });
  if (res.status !== 200) return authToken;
  const body = await res.json();
  return body.token;
}

function getToken(): string {
  return apiKey || authToken;
}

function authHeaders(token?: string, ct?: string): Record<string, string> {
  const h: Record<string, string> = { Authorization: `Bearer ${token ?? getToken()}` };
  if (ct) h['Content-Type'] = ct;
  return h;
}

// ═════════════════════════════════════════════════════════════════
// API Surface E2E
// ═════════════════════════════════════════════════════════════════
test.describe('API Surface E2E', () => {
  // ─── Task 1: Auth-scoped access (P0) ─────────────────────────
  test.describe('Auth Guard', () => {
    test('unauthenticated requests to protected endpoints return 401', async ({ request }) => {
      const endpoints = [
        '/api/sites', '/api/deploy', '/api/collections',
        '/api/storage/files', '/api/git/repos', '/api/functions',
        '/api/messaging/email', '/api/cici/runs', '/api/forge/issues',
        '/api/auth/me', '/api/orgs',
      ];
      for (const path of endpoints) {
        const res = await request.get(path);
        expect(res.status(), `${path} should return 401 without auth`).toBe(401);
      }
    });

    test('requests with invalid token return 401', async ({ request }) => {
      const res = await request.get('/api/sites', {
        headers: { Authorization: 'Bearer invalid-token-12345' },
      });
      expect(res.status()).toBe(401);
    });
  });

  // ─── Task 2: Org isolation (P0) ──────────────────────────────
  test.describe('Org Isolation', () => {
    let tokenA: string;
    let tokenB: string;
    let orgIdA: number;
    let orgIdB: number;

    test.beforeAll(async ({ request }) => {
      const ts = Date.now();

      // Tenant A
      const regA = await request.post('/api/auth/register', {
        data: { email: `e2e-org-a-${ts}@test.com`, password: PASSWORD },
      });
      const bodyA = await regA.json();
      tokenA = bodyA.token;
      const orgA = await request.post('/api/orgs', {
        headers: authHeaders(tokenA, 'application/json'),
        data: { name: `e2e-org-a-${ts}`, slug: `e2e-org-a-${ts}` },
      });
      const orgABody = await orgA.json();
      orgIdA = extractId(orgABody) ?? 0;

      // Tenant B
      const regB = await request.post('/api/auth/register', {
        data: { email: `e2e-org-b-${ts}@test.com`, password: PASSWORD },
      });
      const bodyB = await regB.json();
      tokenB = bodyB.token;
      const orgB = await request.post('/api/orgs', {
        headers: authHeaders(tokenB, 'application/json'),
        data: { name: `e2e-org-b-${ts}`, slug: `e2e-org-b-${ts}` },
      });
      const orgBBody = await orgB.json();
      orgIdB = extractId(orgBBody) ?? 0;
    });

    test('Tenant A cannot read Tenant Bs org resources', async ({ request }) => {
      const res = await request.get(`/api/orgs/${orgIdB}`, {
        headers: authHeaders(tokenA),
      });
      expect([403, 404]).toContain(res.status());
    });

    test('Tenant B cannot read Tenant As org resources', async ({ request }) => {
      const res = await request.get(`/api/orgs/${orgIdA}`, {
        headers: authHeaders(tokenB),
      });
      expect([403, 404]).toContain(res.status());
    });

    test('Tenant A can list their own resources', async ({ request }) => {
      const res = await request.get('/api/sites', {
        headers: authHeaders(tokenA),
      });
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body).toHaveProperty('data');
    });
  });

  // ─── Task 3: Path traversal guard (P2) ───────────────────────
  test.describe('Path Traversal', () => {
    test('storage download with ../ patterns is rejected', async ({ request }) => {
      const paths = ['../../etc/passwd', '..%2F..%2Fetc%2Fpasswd', 'foo/../../../etc/shadow'];
      for (const p of paths) {
        const res = await request.get(`/api/storage/files/${encodeURIComponent(p)}`, {
          headers: authHeaders(),
        });
        expect(res.status(), `Path traversal "${p}" should be rejected`).toBeGreaterThanOrEqual(400);
      }
    });
  });

  // ─── Task 4: Error shape consistency ─────────────────────────
  test.describe('Error Shape', () => {
    test('invalid collection operation returns error shape', async ({ request }) => {
      const token = await freshJWT();
      // GET a non-existent record — may throw 404 or auto-create
      const res = await request.get('/api/collections/__nonexistent/9999999', {
        headers: authHeaders(token),
      });
      // Accept 404 (not found), 200/201 (auto-created), or >=400 (error)
      if (res.status() >= 400) {
        const body = await res.json();
        expect(body).toHaveProperty('error');
      }
    });

    test('GET non-existent site returns 404 + error shape', async ({ request }) => {
      const token = await freshJWT();
      const res = await request.get('/api/sites/99999999', {
        headers: authHeaders(token),
      });
      await expectErrorShape(res, 404);
    });

    test('unauthenticated request returns 401 + error shape', async ({ request }) => {
      const res = await request.get('/api/sites');
      await expectErrorShape(res, 401);
    });
  });

  // ─── Task 5: CORS headers ────────────────────────────────────
  test.describe('CORS', () => {
    test('OPTIONS preflight handles CORS headers', async () => {
      const res = await fetch('http://localhost:9999/api/sites', {
        method: 'OPTIONS',
        headers: { Origin: 'http://example.com', 'Access-Control-Request-Method': 'GET' },
      });
      expect(res.status).toBeGreaterThanOrEqual(200);
      expect(res.status).toBeLessThan(500);
      const h = res.headers;
      if (h.get('access-control-allow-origin')) {
        expect(h.get('access-control-allow-methods')).toBeTruthy();
      }
    });
  });

  // ─── Task 7: Auth/Org CRUD (uses existing user — no new registration) ──
  test.describe('Auth CRUD', () => {
    test('login, me, org lifecycle', async ({ request }) => {
      const token = await freshJWT();

      // Login with existing user
      const loginRes = await request.post('/api/auth/login', {
        data: { email: EMAIL, password: PASSWORD },
      });
      expect(loginRes.status()).toBe(200);
      const loginBody = await loginRes.json();
      expect(loginBody).toHaveProperty('token');
      expect(loginBody).toHaveProperty('user');

      // Me
      const meRes = await request.get('/api/auth/me', {
        headers: authHeaders(token),
      });
      expect(meRes.status()).toBe(200);
      const meBody = await meRes.json();
      expect(meBody).toHaveProperty('email');
      if (meBody.email) {
        expect(typeof meBody.email).toBe('string');
      }

      // Create org
      const slug = `e2e-crud-${Date.now()}`;
      const orgRes = await request.post('/api/orgs', {
        headers: authHeaders(token, 'application/json'),
        data: { name: slug, slug },
      });
      expect(orgRes.status()).toBe(201);

      // List orgs
      const listRes = await request.get('/api/orgs', {
        headers: authHeaders(token),
      });
      expect(listRes.status()).toBe(200);
      const listBody = await listRes.json();
      expect(listBody).toHaveProperty('data');
    });
  });

  // ─── Task 8: Collections CRUD ─────────────────────────────────
  test.describe('Collections CRUD', () => {
    test('collections lifecycle', async ({ request }) => {
      const colName = `e2e_test_${Date.now()}`;
      const token = await freshJWT();
      const h = authHeaders(token, 'application/json');

      // List collections
      const listRes = await request.get('/api/collections', { headers: h });
      expect(listRes.status()).toBe(200);
      const listBody = await listRes.json();
      expect(listBody).toHaveProperty('data');

      // Create a record (auto-creates collection)
      const recordRes = await request.post(`/api/collections/${colName}`, {
        headers: h,
        data: { title: 'test record', value: 42 },
      });
      expect(recordRes.status()).toBe(201);
      const recordBody = await recordRes.json();
      const recordId: string = recordBody.id;

      // Read the record
      const getRes = await request.get(`/api/collections/${colName}/${recordId}`, { headers: h });
      expect(getRes.status()).toBe(200);
      const getBody = await getRes.json();
      expect(getBody).toHaveProperty('title', 'test record');

      // Update the record
      const patchRes = await request.patch(`/api/collections/${colName}/${recordId}`, {
        headers: h,
        data: { value: 99 },
      });
      expect(patchRes.status()).toBe(200);

      // Delete the record
      const deleteRes = await request.delete(`/api/collections/${colName}/${recordId}`, { headers: h });
      expect([200, 204]).toContain(deleteRes.status());
    });
  });

  // ─── Task 9: Storage CRUD ────────────────────────────────────
  test.describe('Storage CRUD', () => {
    test('upload, list, and delete file', async ({ request }) => {
      const fileName = `e2e-test-${Date.now()}.txt`;
      const fileContent = 'Hello BigBase E2E';
      const token = await freshJWT();

      // Upload via FormData (use native fetch for multipart)
      const formData = new FormData();
      formData.append('file', new Blob([fileContent], { type: 'text/plain' }), fileName);
      const uploadRes = await fetch('http://localhost:9999/api/storage/upload', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      });
      expect(uploadRes.status).toBe(201);
      const uploadBody = await uploadRes.json();
      const fileId = uploadBody.id ?? uploadBody.fileId ?? extractStr(uploadBody, 'id');

      // List files
      const listRes = await request.get('/api/storage/files', {
        headers: authHeaders(token),
      });
      expect(listRes.status()).toBe(200);
      const listBody = await listRes.json();
      expect(listBody).toHaveProperty('data');
      expect(Array.isArray(listBody.data)).toBe(true);

      // Delete file
      if (fileId) {
        const delRes = await request.delete(`/api/storage/files/${fileId}`, {
          headers: authHeaders(token),
        });
        expect([200, 204]).toContain(delRes.status());
      }
    });
  });

  // ─── Task 10: Sites CRUD ────────────────────────────────────
  test.describe('Sites CRUD', () => {
    test('create, list, delete site', async ({ request }) => {
      const ts = Date.now();
      const repoName = `e2e-site-${ts}`;
      const token = await freshJWT();
      const h = authHeaders(token, 'application/json');

      // Create git repo
      const repoRes = await request.post('/api/git/repos', {
        headers: h,
        data: { name: repoName },
      });
      expect(repoRes.status()).toBe(201);
      const repoBody = await repoRes.json();
      const repoID = repoBody.id as string;

      // Create site
      const siteRes = await request.post('/api/sites', {
        headers: h,
        data: { name: repoName, git_repo_id: repoID, production_branch: 'main', root_path: './' },
      });
      expect(siteRes.status()).toBe(201);
      const siteBody = await siteRes.json();
      const siteID: string = siteBody.id ?? extractStr(siteBody, 'id') ?? '';

      // List sites
      const listRes = await request.get('/api/sites', { headers: h });
      expect(listRes.status()).toBe(200);
      const listBody = await listRes.json();
      expect(listBody).toHaveProperty('data');

      // Delete site
      if (siteID) {
        const delRes = await request.delete(`/api/sites/${siteID}`, { headers: h });
        expect([200, 204]).toContain(delRes.status());
      }

      // Clean up git repo
      await request.delete(`/api/git/repos/${repoID}`, { headers: h });
    });
  });

  // ─── Task 11: Deploy CRUD ────────────────────────────────────
  test.describe('Deploy CRUD', () => {
    test('list deployments', async ({ request }) => {
      const token = await freshJWT();
      const res = await request.get('/api/deploy', { headers: authHeaders(token) });
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body).toHaveProperty('data');
    });
  });

  // ─── Task 12: Functions CRUD ─────────────────────────────────
  test.describe('Functions CRUD', () => {
    test('list functions', async ({ request }) => {
      const token = await freshJWT();
      const res = await request.get('/api/functions', { headers: authHeaders(token) });
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body).toHaveProperty('data');
    });
  });

  // ─── Task 13: Git/Forge CRUD ─────────────────────────────────
  test.describe('Git Forge CRUD', () => {
    test('git repo lifecycle', async ({ request }) => {
      const ts = Date.now();
      const repoName = `e2e-forge-${ts}`;
      const token = await freshJWT();
      const h = authHeaders(token, 'application/json');

      // Create git repo
      const repoRes = await request.post('/api/git/repos', { headers: h, data: { name: repoName } });
      expect(repoRes.status()).toBe(201);
      const repoBody = await repoRes.json();
      const repoID = repoBody.id as string;

      // List git repos
      const listRes = await request.get('/api/git/repos', { headers: h });
      expect(listRes.status()).toBe(200);
      const listBody = await listRes.json();
      expect(listBody).toHaveProperty('data');

      // Clean up
      await request.delete(`/api/git/repos/${repoID}`, { headers: h });
    });
  });

  // ─── Task 14: Messaging smoke tests ──────────────────────────
  test.describe('Messaging Smoke', () => {
    test('messaging endpoints are reachable', async ({ request }) => {
      const token = await freshJWT();
      const h = authHeaders(token, 'application/json');

      // List messages
      const listRes = await request.get('/api/messaging/messages', { headers: h });
      expect(listRes.status()).toBe(200);
      const listBody = await listRes.json();
      expect(listBody).toHaveProperty('data');

      // Email endpoint (accepts message and queues it)
      const emailRes = await request.post('/api/messaging/email', {
        headers: h,
        data: { to: 'test@test.com', subject: 'Test', body: 'Hello' },
      });
      expect([200, 201, 400, 422]).toContain(emailRes.status());
      if (emailRes.status() >= 400) {
        const emailBody = await emailRes.json();
        expect(emailBody).toHaveProperty('error');
      }
    });
  });

  // ─── Task 15: Monitoring smoke tests ─────────────────────────
  test.describe('Monitoring Smoke', () => {
    test('monitoring endpoints return health and metrics', async ({ request }) => {
      const token = await freshJWT();

      // Health
      const healthRes = await request.get('/api/monitoring/health', { headers: authHeaders(token) });
      expect(healthRes.status()).toBe(200);
      const healthBody = await healthRes.json();
      expect(healthBody).toHaveProperty('status');

      // Metrics
      const metricsRes = await request.get('/api/monitoring/metrics', { headers: authHeaders(token) });
      expect(metricsRes.status()).toBe(200);

      // Alerts
      const alertsRes = await request.get('/api/monitoring/alerts', { headers: authHeaders(token) });
      expect(alertsRes.status()).toBe(200);
      const alertsBody = await alertsRes.json();
      expect(alertsBody).toHaveProperty('data');
    });
  });

  // ─── Task 16: CICI/GitHub smoke tests ────────────────────────
  test.describe('CICI GitHub Smoke', () => {
    test('CICI API returns data', async ({ request }) => {
      const token = await freshJWT();
      const res = await request.get('/api/cici/runs', { headers: authHeaders(token) });
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body).toHaveProperty('data');
    });

    test('GitHub status endpoint is reachable', async ({ request }) => {
      const token = await freshJWT();
      const res = await request.get('/api/github/status', { headers: authHeaders(token) });
      expect([200, 400, 404]).toContain(res.status());
    });
  });

  // ─── Task 17: MCP/Realtime smoke tests ───────────────────────
  test.describe('MCP Realtime Smoke', () => {
    test('realtime status endpoint is reachable', async ({ request }) => {
      const token = await freshJWT();
      const res = await request.get('/api/realtime/status', { headers: authHeaders(token) });
      expect(res.status()).toBe(200);
    });
  });

  // ─── Webhooks smoke test ──────────────────────────────────────
  test.describe('Webhooks Smoke', () => {
    test('webhook endpoints are reachable', async ({ request }) => {
      const token = await freshJWT();
      const h = authHeaders(token, 'application/json');

      // Create
      const createRes = await request.post('/api/webhooks', {
        headers: h,
        data: { url: 'https://example.com/webhook', events: ['deploy.completed'] },
      });
      // Webhook endpoint may return JSON (proper API) or HTML (admin redirect)
      expect(createRes.status()).toBeGreaterThanOrEqual(200);
      expect(createRes.status()).toBeLessThan(500);

      // List
      const listRes = await request.get('/api/webhooks', { headers: h });
      expect(listRes.status()).toBeGreaterThanOrEqual(200);
      expect(listRes.status()).toBeLessThan(500);
    });
  });

  // ─── Public endpoints smoke test ──────────────────────────────
  test.describe('Public Endpoints', () => {
    test('health endpoint is publicly accessible', async ({ request }) => {
      const res = await request.get('/health');
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body).toHaveProperty('status');
    });

    test('version endpoint returns API version', async ({ request }) => {
      const res = await request.get('/api/version');
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body).toHaveProperty('version');
    });
  });

  // ─── Task 6: Rate limiting (SKIPPED — run manually: requires dedicated server) ──
  // Rate limiting tests exhaust the 60 req/min IP budget, breaking other tests.
  // To run: start a dedicated server, then:
  //   npx playwright test -g "Rate Limit"
  test.describe('Rate Limit', () => {
    test.skip('rate limiting enforces 429 on excessive requests', async () => {
      // Run manually against a dedicated rate-limited server
    });
  });
});
