/**
 * Helpers for the admin / profile / pages / responsive specs.
 * Kept separate from fixtures/index.ts, which other specs share.
 */
import {
  expect,
  test,
  request as playwrightRequest,
  type APIRequestContext,
  type Browser,
  type BrowserContext,
  type Locator,
  type Page
} from '@playwright/test'
import { mkdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { ADMIN_STATE, apiHeaders } from './env'
import { newStudent, registerStudent, stopActiveSession, type Student } from './index'

const E2E_DIR = join(dirname(fileURLToPath(import.meta.url)), '..')

export const FIRST_TASK = 'nginx-health'
export const TASK_COUNT = 6

// ---------------------------------------------------------------------------
// API contexts
// ---------------------------------------------------------------------------

/** A standalone API context authenticated as the seeded admin. Dispose it when done. */
export async function adminApi(baseURL: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({ baseURL, storageState: ADMIN_STATE, extraHTTPHeaders: apiHeaders(baseURL) })
}

export interface AdminUserRow {
  id: number
  username: string
  email: string
  role: 'student' | 'admin'
  active: boolean
}

export async function findUser(api: APIRequestContext, username: string): Promise<AdminUserRow | undefined> {
  const res = await api.get('/api/admin/users')
  expect(res.status()).toBe(200)
  const users = (await res.json()) as AdminUserRow[]
  return users.find((u) => u.username === username)
}

/** Starts a lab session through the API and returns its id. */
export async function startSession(api: APIRequestContext, taskId = FIRST_TASK): Promise<string> {
  const res = await api.post('/api/sessions', { data: { taskId } })
  expect(res.status(), await res.text()).toBe(201)
  return ((await res.json()) as { sessionId: string }).sessionId
}

// ---------------------------------------------------------------------------
// Extra student in its own browser context
// ---------------------------------------------------------------------------

export interface StudentContext {
  student: Student
  context: BrowserContext
  page: Page
  /** Shares cookies with `page`. */
  api: APIRequestContext
  /** Stops the student's lab session (best effort) and closes the context. */
  close: () => Promise<void>
}

/**
 * Registers a fresh student in a separate browser context (cookie lives in that context only).
 * `contextOptions` lets callers pass the project's device settings.
 */
export async function newStudentContext(
  browser: Browser,
  baseURL: string,
  prefix = 'e2e',
  contextOptions: Parameters<Browser['newContext']>[0] = {}
): Promise<StudentContext> {
  const context = await browser.newContext({ ...contextOptions, baseURL, extraHTTPHeaders: apiHeaders(baseURL) })
  const page = await context.newPage()
  const student = newStudent(prefix)
  await registerStudent(context.request, student)
  return {
    student,
    context,
    page,
    api: context.request,
    close: async () => {
      await stopActiveSession(context.request).catch(() => undefined)
      await context.close()
    }
  }
}

// ---------------------------------------------------------------------------
// Page health: console errors, failed requests, horizontal scroll
// ---------------------------------------------------------------------------

export interface PageProblems {
  consoleErrors: string[]
  serverErrors: string[]
}

/**
 * Collects console errors, uncaught page errors and responses with status >= 500.
 * `allowUnauthenticatedMe` ignores the 401 of the GET /api/me bootstrap (logged-out pages only).
 *
 * nginx 429s (per-IP `api` zone, 20 r/s) are environment noise here: every e2e browser shares one
 * client IP. They are not counted as console errors but are recorded as a test annotation.
 */
export function watchPage(page: Page, opts: { allowUnauthenticatedMe?: boolean } = {}): PageProblems {
  const problems: PageProblems = { consoleErrors: [], serverErrors: [] }
  let meUnauthorized = 0
  const throttled = new Set<string>()
  page.on('response', (res) => {
    if (res.status() >= 500) problems.serverErrors.push(`${res.status()} ${res.request().method()} ${res.url()}`)
    if (res.status() === 401 && new URL(res.url()).pathname === '/api/me') meUnauthorized += 1
    if (res.status() === 429) {
      throttled.add(res.url())
      test.info().annotations.push({ type: 'rate-limited (429)', description: `${res.request().method()} ${res.url()}` })
    }
  })
  page.on('pageerror', (err) => problems.consoleErrors.push(`pageerror: ${err.message}`))
  page.on('console', (msg) => {
    if (msg.type() !== 'error') return
    const text = msg.text()
    const loc = msg.location()?.url ?? ''
    if (/status of 429/.test(text) && throttled.has(loc)) return
    if (
      opts.allowUnauthenticatedMe &&
      meUnauthorized > 0 &&
      /status of 401/.test(text) &&
      (loc === '' || new URL(loc, 'http://x').pathname === '/api/me')
    ) {
      return
    }
    problems.consoleErrors.push(`console: ${text}${loc ? ` (${loc})` : ''}`)
  })
  return problems
}

export function expectClean(problems: PageProblems, where: string): void {
  expect(problems.consoleErrors, `console errors on ${where}`).toEqual([])
  expect(problems.serverErrors, `5xx responses on ${where}`).toEqual([])
}

export async function horizontalOverflow(page: Page): Promise<{ scrollWidth: number; innerWidth: number }> {
  return page.evaluate(() => ({ scrollWidth: document.documentElement.scrollWidth, innerWidth: window.innerWidth }))
}

export async function expectNoHorizontalScroll(page: Page, where: string): Promise<void> {
  const { scrollWidth, innerWidth } = await horizontalOverflow(page)
  expect(scrollWidth, `horizontal page scroll on ${where} (scrollWidth ${scrollWidth} > innerWidth ${innerWidth})`).toBeLessThanOrEqual(
    innerWidth
  )
}

/** Fails when any section-level error state (EmptyState tone="danger" renders role="alert") is visible. */
export async function expectNoErrorState(page: Page, where: string): Promise<void> {
  const alerts = page.locator('[role="alert"]').filter({ hasText: /yuklab bo'lmadi|xatolik/i })
  await expect(alerts, `error state on ${where}`).toHaveCount(0)
}

/** Every <table> must sit inside a horizontally scrollable container that fits the viewport. */
export async function expectTablesScrollable(page: Page, where: string): Promise<void> {
  const report = await page.evaluate(() => {
    const out: string[] = []
    for (const table of Array.from(document.querySelectorAll('table'))) {
      if (table.classList.contains('sr-only') || table.getBoundingClientRect().width === 0) continue
      let el: HTMLElement | null = table.parentElement
      let scroller: HTMLElement | null = null
      while (el && el !== document.body) {
        const ox = getComputedStyle(el).overflowX
        if (ox === 'auto' || ox === 'scroll') {
          scroller = el
          break
        }
        el = el.parentElement
      }
      const label = table.querySelector('caption')?.textContent?.trim() ?? 'table'
      if (!scroller) out.push(`${label}: no scrollable ancestor`)
      else if (scroller.getBoundingClientRect().right > window.innerWidth + 1) out.push(`${label}: scroller wider than viewport`)
    }
    return out
  })
  expect(report, `tables on ${where}`).toEqual([])
}

/** Visible, enabled and at least 40px tall (comfortable tap target). */
export async function expectTappable(locator: Locator, name: string): Promise<void> {
  await expect(locator, `${name} visible`).toBeVisible()
  await locator.scrollIntoViewIfNeeded()
  const box = await locator.boundingBox()
  expect(box, `${name} has a box`).not.toBeNull()
  expect(box!.height, `${name} tap height`).toBeGreaterThanOrEqual(40)
  const vw = locator.page().viewportSize()?.width ?? 0
  expect(box!.x, `${name} inside viewport (left)`).toBeGreaterThanOrEqual(-1)
  expect(box!.x + box!.width, `${name} inside viewport (right)`).toBeLessThanOrEqual(vw + 1)
}

// ---------------------------------------------------------------------------
// Screenshots (opt-in: CAPTURE_SCREENSHOTS=1)
// ---------------------------------------------------------------------------

export const CAPTURE = process.env.CAPTURE_SCREENSHOTS === '1'

export async function capture(
  p: Page,
  kind: 'desktop' | 'mobile',
  name: string,
  opts: { fullPage?: boolean } = {}
): Promise<void> {
  if (!CAPTURE) return
  const dir = join(E2E_DIR, 'screenshots', kind)
  mkdirSync(dir, { recursive: true })
  // Let entrance transitions and toasts settle so the image is stable.
  await p.waitForTimeout(400)
  await p.screenshot({ path: join(dir, `${name}.png`), fullPage: opts.fullPage ?? true, animations: 'disabled', caret: 'hide' })
}

// ---------------------------------------------------------------------------
// Lab terminal
// ---------------------------------------------------------------------------

export function terminalRows(p: Page) {
  return p.locator('.xterm-rows')
}

/** Waits until the WebSocket shell answers (output that is not an echo of the typed command). */
export async function waitForShell(p: Page): Promise<void> {
  await expect(terminalRows(p)).toBeVisible({ timeout: 30_000 })
  await expect(async () => {
    await p.locator('.xterm-host').click()
    await p.keyboard.type('echo E2E_$((40+2))')
    await p.keyboard.press('Enter')
    await expect(terminalRows(p)).toContainText('E2E_42', { timeout: 3_000 })
  }).toPass({ timeout: 45_000 })
}

/** Types a command; when `expectOutput` is given, waits for it to appear in the terminal. */
export async function runInTerminal(p: Page, command: string, expectOutput?: string | RegExp, timeout = 30_000): Promise<void> {
  await p.locator('.xterm-host').click()
  await p.keyboard.type(command, { delay: 5 })
  await p.keyboard.press('Enter')
  if (expectOutput !== undefined) await expect(terminalRows(p)).toContainText(expectOutput, { timeout })
}

/** Commands of the reference solution (comments and shell options stripped). */
export function solutionCommands(taskId = FIRST_TASK): string[] {
  const script = readFileSync(join(E2E_DIR, 'fixtures', 'solutions', `${taskId}.sh`), 'utf8')
  return script
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('#') && !l.startsWith('set '))
}

/**
 * Types the reference solution into the terminal, one command at a time, waiting for a marker after
 * each so commands do not interleave.
 */
export async function typeSolution(p: Page, taskId = FIRST_TASK): Promise<void> {
  for (const [i, cmd] of solutionCommands(taskId).entries()) {
    await runInTerminal(p, `${cmd}; echo STEP_$((${i}+100))_DONE`, `STEP_${i + 100}_DONE`, 60_000)
  }
}

/** Starts the lab for `taskId` from the start screen and waits for a working shell. */
export async function openLab(p: Page, taskId = FIRST_TASK): Promise<void> {
  await p.goto(`/lab/${taskId}`)
  const start = p.getByRole('button', { name: 'Muhitni ishga tushirish' })
  const terminal = p.locator('section[aria-label="Terminal"] .xterm')
  await expect(start.or(terminal)).toBeVisible({ timeout: 30_000 })
  if (await start.isVisible()) await start.click()
  await waitForShell(p)
}

/** Clicks "Yechimni tekshirish" in the terminal pane and waits for the score line. */
export async function runCheck(p: Page): Promise<string> {
  await p.locator('section[aria-label="Terminal"]').getByRole('button', { name: /Yechimni tekshirish/ }).click()
  // Vue condenses template whitespace into single spaces, so do not anchor the pattern.
  const score = p.getByRole('status').filter({ hasText: /\d+\/\d+ passed/ }).locator('span').first()
  await expect(score).toHaveText(/\d+\/\d+ passed/, { timeout: 60_000 })
  return (await score.textContent())?.trim() ?? ''
}

// ---------------------------------------------------------------------------
// UI helpers
// ---------------------------------------------------------------------------

/** The open ConfirmDialog. */
export function dialog(p: Page) {
  return p.getByRole('dialog')
}

export function toast(p: Page, text: string | RegExp) {
  return p.locator('[aria-live="polite"][role="status"]').filter({ hasText: text }).getByText(text)
}

/** UI login through the /login form. */
export async function uiLogin(p: Page, username: string, password: string): Promise<void> {
  await p.getByLabel('Foydalanuvchi nomi').fill(username)
  await p.getByLabel('Parol', { exact: true }).fill(password)
  await p.getByRole('button', { name: 'Kirish', exact: true }).click()
}
