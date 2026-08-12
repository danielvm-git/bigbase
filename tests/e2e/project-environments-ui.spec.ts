// story: e89s02
import { test, expect } from '@playwright/test';

test.describe('Project and Environment navigation', () => {
  test('unauthenticated Project access is denied', async ({ request }) => {
    const response = await request.get('/api/projects');
    expect(response.status()).toBe(401);
  });

  test('authenticated operator creates a Project and Environment', async ({ request }) => {
    const email = `e89-project-${Date.now()}@test.com`;
    const registration = await request.post('/api/auth/register', {
      data: { email, password: 'TestPass123!' },
    });
    expect(registration.status()).toBe(201);
    const registrationBody = await registration.json();
    const headers = { Authorization: `Bearer ${registrationBody.token}` };

    const projectResponse = await request.post('/api/projects', {
      headers,
      data: { name: `e89-project-${Date.now()}` },
    });
    expect(projectResponse.status()).toBe(201);
    const projectBody = await projectResponse.json();
    const project = projectBody.data;

    const environmentResponse = await request.post(`/api/projects/${project.id}/environments`, {
      headers,
      data: { slug: 'staging', name: 'Staging' },
    });
    expect(environmentResponse.status()).toBe(201);
    const environmentBody = await environmentResponse.json();
    expect(environmentBody.data).toMatchObject({
      project_id: project.id,
      slug: 'staging',
      name: 'Staging',
    });
  });
});
