// story: e87s03 (WCAG 2.2 AAA 2.5.5 — pointer targets ≥44×44 CSS px)
//
// Measures bounding boxes of interactive targets across the app's main
// routes and asserts every target is ≥44×44 OR matches a documented 2.5.5
// exception. Prints the target→size→exception inventory per route.
//
// Exception map (recorded in specs/epics/e87-wcag-aaa/e87s03-spec.md):
//   - .actions-cell children → spacing exception (row-level table text actions)
//   - .btn-compact           → explicit density opt-out (same spacing exception)
//   - .btn-link              → inline exception (text links within prose)
//   - .breadcrumb a          → inline exception (breadcrumb text links)
//   - .skip-link             → offscreen focus helper (not a pointer target)
//
// Form controls (input/select/textarea) are excluded from the inventory:
// the e87 scope doc assessed Input/Select target height as acceptable and
// they are not part of this story's target classes.
import { test, expect, type Page } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-target-size-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
const MIN_TARGET = 44;

const TARGET_SELECTOR = [
  'button',
  'a[href]',
  '[role="button"]',
  '[role="tab"]',
  '[role="menuitem"]',
  '[role="menuitemradio"]',
].join(', ');

interface TargetEntry {
  route: string
  name: string
  tag: string
  classes: string
  width: number
  height: number
  status: 'ok' | 'exception' | 'FAIL'
  reason: string
}

async function collectTargets(page: Page, route: string): Promise<TargetEntry[]> {
  return page.evaluate(
    ({ selector, min, route }) => {
      const exceptionOf = (el: Element): string | null => {
        if (el.closest('.skip-link')) return 'skip-link (offscreen focus helper)'
        if (el.closest('.actions-cell')) return 'spacing exception: row-level table text action (.actions-cell)'
        if (el.classList.contains('btn-compact')) return 'spacing exception: explicit density opt-out (.btn-compact)'
        if (el.classList.contains('btn-link')) return 'inline exception: text link (.btn-link)'
        if (el.closest('.breadcrumb')) return 'inline exception: breadcrumb text link'
        if (el.closest('.app-footer')) return 'inline exception: footer text link'
        return null
      }
      const items: TargetEntry[] = []
      for (const el of document.querySelectorAll<HTMLElement>(selector)) {
        const style = getComputedStyle(el)
        if (style.display === 'none' || style.visibility === 'hidden' || el.getAttribute('aria-hidden') === 'true') continue
        if (el.closest('.skip-link')) continue
        const rect = el.getBoundingClientRect()
        if (rect.width < 1 || rect.height < 1) continue
        const reason = exceptionOf(el)
        const pass = rect.width >= min && rect.height >= min
        items.push({
          route,
          name: (el.getAttribute('aria-label') || el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 40),
          tag: el.tagName.toLowerCase(),
          classes: typeof el.className === 'string' ? el.className.slice(0, 60) : '',
          width: Math.round(rect.width),
          height: Math.round(rect.height),
          status: pass ? 'ok' : reason ? 'exception' : 'FAIL',
          reason: pass ? '' : (reason ?? ''),
        })
      }
      return items
    },
    { selector: TARGET_SELECTOR, min: MIN_TARGET, route },
  )
}

function assertInventory(items: TargetEntry[], context: string): void {
  const failures = items.filter(i => i.status === 'FAIL')
  // eslint-disable-next-line no-console
  console.log(`\n=== ${context} — ${items.length} targets (${failures.length} failing) ===`)
  for (const it of items) {
    // eslint-disable-next-line no-console
    console.log(
      `  ${it.status.padEnd(9)} ${String(it.width).padStart(3)}×${String(it.height).padStart(3)}  ` +
        `<${it.tag}> .${it.classes.replace(/ /g, ' .')}  "${it.name}"  ${it.reason}`,
    )
  }
  expect(
    failures.map(f => `${f.route}: <${f.tag}> "${f.name}" (${f.classes}) = ${f.width}×${f.height}`),
    `${context}: targets below 44px without a documented exception`,
  ).toEqual([])
}

let token = ''
// The e2e server shares /tmp/bigbase-e2e.db across runs (and worktrees), so
// collection tables and admin roles persist. Use a run-unique collection name
// so Data Studio always renders rows → schema columns → action cells.
const COLLECTION = `targets_${Date.now()}`

test.beforeAll(async ({ request }) => {
  const reg = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  })
  expect(reg.status()).toBe(201)
  token = ((await reg.json()) as { token: string }).token

  // Seed a run-unique collection (table + record) and a storage file so
  // table action cells render.
  const col = await request.post(`/api/collections/${COLLECTION}`, {
    data: { name: 'probe' },
  })
  expect([200, 201, 409]).toContain(col.status())
  const up = await request.post('/api/storage/upload', {
    multipart: {
      file: { name: 'probe.txt', mimeType: 'text/plain', buffer: Buffer.from('target-size probe') },
    },
  })
  expect([200, 201]).toContain(up.status())
})

test.beforeEach(async ({ context }) => {
  await context.addCookies([{ name: 'token', value: token, url: 'http://localhost:9999' }])
})

test('app shell: dashboard, sidebar nav, theme picker targets ≥44px', async ({ page }) => {
  await page.goto('/admin/#/')
  await page.waitForSelector('.sidebar-nav a', { timeout: 15000 })

  const items = await collectTargets(page, 'Dashboard')
  await page.click('.theme-trigger')
  await page.waitForSelector('.theme-menu-item', { timeout: 5000 })
  items.push(...(await collectTargets(page, 'Theme menu')))
  await page.keyboard.press('Escape')

  assertInventory(items, 'App shell (dashboard + sidebar + theme menu)')
})

test('data studio: collection nav, mode toggles, actions cells, modal close', async ({ page }) => {
  await page.goto('/admin/#/data')
  await page.waitForSelector('.collection-btn', { timeout: 15000 })
  await page.click(`.collection-btn:has-text("${COLLECTION}")`)
  await page.waitForSelector('.studio-mode-toggle', { timeout: 15000 })
  await page.getByRole('button', { name: 'Schema', exact: true }).click()
  await page.waitForSelector('.actions-cell', { timeout: 15000 })

  const items = await collectTargets(page, 'Data Studio (schema)')
  await page.getByRole('button', { name: 'Add column' }).click()
  await page.waitForSelector('.modal-close', { timeout: 5000 })
  items.push(...(await collectTargets(page, 'Data Studio (modal)')))
  await page.keyboard.press('Escape')

  assertInventory(items, 'Data Studio (schema + modal)')
})

test('storage + users tables: action cells and row actions', async ({ page }) => {
  await page.goto('/admin/#/storage')
  await page.waitForSelector('.actions-cell', { timeout: 15000 })
  const items = await collectTargets(page, 'Storage (actions cell)')

  await page.goto('/admin/#/users')
  // /api/auth/users requires admin; on the shared /tmp e2e DB the first
  // registered user already owns admin, so a later run's fresh user gets 403.
  // When the table renders, measure the row Delete buttons; otherwise skip
  // (row Delete is a plain .btn-sm, covered by the button sizing rules).
  const usersRendered = await page
    .waitForSelector('tbody tr', { timeout: 10000 })
    .then(() => true)
    .catch(() => false)
  if (usersRendered) {
    items.push(...(await collectTargets(page, 'Users table')))
  } else {
    // eslint-disable-next-line no-console
    console.log('  (users route skipped: fresh e2e user lacks admin role on the shared DB)')
  }

  assertInventory(items, 'Storage + Users tables')
})

test('sites tabs, monitoring tabs, login targets', async ({ page }) => {
  await page.goto('/admin/#/deploy')
  await page.waitForSelector('.segmented-control, .empty-state, .site-grid', { timeout: 15000 })
  const items = await collectTargets(page, 'Sites')

  await page.goto('/admin/#/monitoring')
  await page.waitForSelector('.tab, .tabs', { timeout: 15000 })
  items.push(...(await collectTargets(page, 'Monitoring')))

  await page.goto('/admin/#/login')
  await page.waitForSelector('.google-btn, .btn', { timeout: 15000 })
  items.push(...(await collectTargets(page, 'Login')))

  assertInventory(items, 'Sites + Monitoring + Login')
})

test('mobile: sidebar toggle and collapsed icon nav targets ≥44px', async ({ browser }) => {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 } })
  await context.addCookies([{ name: 'token', value: token, url: 'http://localhost:9999' }])
  const page = await context.newPage()
  await page.goto('/admin/#/')
  await page.waitForSelector('.sidebar-toggle', { timeout: 15000 })

  const items = await collectTargets(page, 'Mobile shell')
  assertInventory(items, 'Mobile shell (sidebar toggle + collapsed nav)')
  await context.close()
})
