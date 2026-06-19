import { test, expect } from '@playwright/test';

const email = `e2e-sites-${Date.now()}@test.com`;
const password = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  // Register and login to get auth token
  await request.post('/api/auth/register', {
    data: { email, password },
  });

  const res = await request.post('/api/auth/login', {
    data: { email, password },
  });
  const body = await res.json();
  authToken = body.token;
});

test('GET /api/sites returns 200 with data array', async ({ request }) => {
  const res = await request.get('/api/sites', {
    headers: { Authorization: `Bearer ${authToken}` },
  });
  expect(res.status()).toBe(200);

  const body = await res.json();
  expect(body).toHaveProperty('data');
  expect(Array.isArray(body.data)).toBe(true);
});

test('POST /api/sites returns 400 when required fields missing', async ({ request }) => {
  const res = await request.post('/api/sites', {
    headers: { Authorization: `Bearer ${authToken}` },
    data: { name: 'e2e-test-site' },
  });
  expect(res.status()).toBe(400);

  const body = await res.json();
  expect(body).toHaveProperty('error');
});

test('GET /api/deploy returns 200 with data array', async ({ request }) => {
  const res = await request.get('/api/deploy', {
    headers: { Authorization: `Bearer ${authToken}` },
  });
  expect(res.status()).toBe(200);

  const body = await res.json();
  expect(body).toHaveProperty('data');
  expect(Array.isArray(body.data)).toBe(true);
});

test('delete site removes site from list', async ({ request }) => {
  // Create a git repo first
  const repoName = `e2e-delete-${Date.now()}`
  const repoRes = await request.post('/api/git/repos', {
    headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
    data: { name: repoName },
  })
  expect(repoRes.status()).toBe(201)
  const repoBody = await repoRes.json()
  const repoID = repoBody.id as string

  // Create a site from that repo
  const siteRes = await request.post('/api/sites', {
    headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
    data: { name: repoName, git_repo_id: repoID, production_branch: 'main', root_path: './' },
  })
  expect(siteRes.status()).toBe(201)
  const siteBody = await siteRes.json()
  const siteID = siteBody.id as string

  // Delete the site
  const delRes = await request.delete(`/api/sites/${siteID}`, {
    headers: { Authorization: `Bearer ${authToken}` },
  })
  expect(delRes.status()).toBe(204)

  // Verify the site no longer appears in GET /api/sites/:id
  const getRes = await request.get(`/api/sites/${siteID}`, {
    headers: { Authorization: `Bearer ${authToken}` },
  })
  expect(getRes.status()).toBe(404)

  // Verify the site is absent from the list
  const listRes = await request.get('/api/sites', {
    headers: { Authorization: `Bearer ${authToken}` },
  })
  expect(listRes.status()).toBe(200)
  const listBody = await listRes.json()
  const found = (listBody.data as { id: string }[]).some((s: { id: string }) => s.id === siteID)
  expect(found).toBe(false)

  // Clean up the git repo
  await request.delete(`/api/git/repos/${repoID}`, {
    headers: { Authorization: `Bearer ${authToken}` },
  })
});
