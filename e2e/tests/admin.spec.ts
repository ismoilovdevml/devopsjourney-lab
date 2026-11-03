import { request as playwrightRequest, type APIRequestContext, type Page } from '@playwright/test'
import { test, expect, newStudent, registerStudent, stopActiveSession, type Student } from '../fixtures'
import { ADMIN_STATE, apiHeaders } from '../fixtures/env'
import {
  TASK_COUNT,
  adminApi,
  dialog,
  expectNoErrorState,
  findUser,
  newStudentContext,
  startSession,
  toast,
  uiLogin
} from '../fixtures/admin'

const INFO_COLOR = 'rgb(61, 220, 151)' // #3DDC97 (accent)
const WARN_COLOR = 'rgb(245, 179, 66)' // #F5B342 (warn)

/** A fresh student that exists only as an API context (no browser page). */
async function apiStudent(baseURL: string, prefix = 'e2e-adm'): Promise<{ student: Student; api: APIRequestContext }> {
  const api = await playwrightRequest.newContext({ baseURL, extraHTTPHeaders: apiHeaders(baseURL) })
  const student = newStudent(prefix)
  await registerStudent(api, student)
  return { student, api }
}

function userRow(page: Page, username: string) {
  return page.getByRole('row').filter({ has: page.getByRole('rowheader', { name: new RegExp(`^${username}\\b`) }) })
}

async function searchUser(page: Page, username: string) {
  await page.goto('/admin?section=users')
  await page.getByPlaceholder('grep user / email').fill(username)
  await expect(userRow(page, username)).toBeVisible()
  await expect(page.getByText(/^1 \/ \d+ ta$/)).toBeVisible()
}

test.describe('admin console', () => {
  // The default `page` is the seeded admin in this file (project viewport), except where noted.
  test.use({ storageState: ADMIN_STATE })

  let admin: APIRequestContext
  test.beforeAll(async ({ baseURL }) => {
    admin = await adminApi(baseURL!)
  })
  test.afterAll(async () => {
    await admin.dispose()
  })

  test('header shows sudo badge and live system status', async ({ page }) => {
    await page.goto('/admin')
    const header = page.getByRole('banner')
    await expect(header.getByText('sudo', { exact: true })).toBeVisible()
    await expect(header.getByRole('status').filter({ hasText: 'all systems operational' })).toBeVisible()
    await expect(page.getByRole('heading', { level: 1, name: "Umumiy ko'rinish" })).toBeVisible()
  })

  test('every section renders real data without an error state', async ({ page }) => {
    // Overview: KPI numbers and the 24h chart.
    await page.goto('/admin?section=overview')
    const main = page.locator('main')
    for (const label of ["O'quvchilar", 'Faol foydalanuvchilar', 'Faol sessiyalar']) {
      const value = main.locator('span').filter({ hasText: new RegExp(`^${label}$`) }).first().locator('xpath=following-sibling::span[1]')
      await expect(value, label).toHaveText(/^\d[\d ]*$/)
    }
    const rate = main.locator('span').filter({ hasText: /^Yakunlash darajasi$/ }).first().locator('xpath=following-sibling::span[1]')
    await expect(rate).toHaveText(/^\d+(\.\d)?%$/)
    const chart = main.locator('[role="img"][aria-label*="24 soat"]')
    await expect(chart.locator(':scope > span')).toHaveCount(24)
    await expect(main.getByRole('progressbar').first()).toBeVisible()
    await expectNoErrorState(page, 'overview')

    // Users: a populated table.
    await page.goto('/admin?section=users')
    await expect(page.getByRole('heading', { level: 1, name: 'Foydalanuvchilar' })).toBeVisible()
    await expect(page.getByText(/^\d+ \/ \d+ ta$/)).toBeVisible()
    await expect(main.locator('tbody tr th[scope="row"]').first()).toBeVisible()
    expect(await main.locator('tbody tr th[scope="row"]').count()).toBeGreaterThan(0)
    await expectNoErrorState(page, 'users')

    // Sessions: table or the empty state, plus the auto-refresh indicator.
    await page.goto('/admin?section=sessions')
    await expect(page.getByRole('heading', { level: 1, name: 'Faol sessiyalar' })).toBeVisible()
    await expect(page.getByText(/auto-refresh 10s/)).toBeVisible()
    await expect(page.getByText(/yangilandi \d\d:\d\d:\d\d/)).toBeVisible()
    await expect(
      page.getByRole('region', { name: 'Faol sessiyalar jadvali' }).or(page.getByText("Faol sessiyalar yo'q"))
    ).toBeVisible()
    await expectNoErrorState(page, 'sessions')

    // Activity: log lines.
    await page.goto('/admin?section=activity')
    await expect(page.getByRole('heading', { level: 1, name: 'Faollik jurnali' })).toBeVisible()
    const entries = page.getByRole('list', { name: 'Faollik yozuvlari' }).getByRole('listitem')
    await expect(entries.first()).toBeVisible()
    await expectNoErrorState(page, 'activity')

    // Resources: container meter with numbers.
    await page.goto('/admin?section=resources')
    await expect(page.getByRole('heading', { level: 1, name: 'Server resurslari' })).toBeVisible()
    const containers = main.locator('span').filter({ hasText: /^containers$/ }).locator('xpath=following-sibling::span[1]')
    await expect(containers).toHaveText(/^\d+\/\d+$/)
    await expect(page.getByRole('meter', { name: 'Ishlayotgan konteynerlar limitga nisbatan' })).toBeVisible()
    await expectNoErrorState(page, 'resources')

    // Tasks: the catalogue.
    await page.goto('/admin?section=tasks')
    await expect(page.getByRole('heading', { level: 1, name: 'Vazifalar' })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Vazifalar katalogi' }).getByRole('listitem')).toHaveCount(TASK_COUNT)
    await expectNoErrorState(page, 'tasks')
  })

  test('users: search finds a new student and role toggles student → admin → student', async ({ page, baseURL }) => {
    const { student, api } = await apiStudent(baseURL!)
    try {
      await searchUser(page, student.username)
      const row = userRow(page, student.username)
      await expect(row.getByText("O'quvchi", { exact: true })).toBeVisible()

      await row.getByRole('button', { name: 'Admin qilish' }).click()
      await expect(dialog(page).getByRole('heading', { name: 'Admin huquqini berish' })).toBeVisible()
      await dialog(page).getByRole('button', { name: 'Admin qilish' }).click()
      await expect(toast(page, `${student.username}: rol "Admin" ga o'zgardi`)).toBeVisible()
      await expect(row.getByText('Admin', { exact: true })).toBeVisible()
      expect((await findUser(admin, student.username))?.role).toBe('admin')

      await row.getByRole('button', { name: "O'quvchi qilish" }).click()
      await expect(dialog(page).getByRole('heading', { name: 'Admin huquqini olish' })).toBeVisible()
      await dialog(page).getByRole('button', { name: "O'quvchi qilish" }).click()
      await expect(toast(page, `${student.username}: rol "O'quvchi" ga o'zgardi`)).toBeVisible()
      await expect(row.getByText("O'quvchi", { exact: true })).toBeVisible()
      expect((await findUser(admin, student.username))?.role).toBe('student')
    } finally {
      await api.dispose()
    }
  })

  test('users: deactivate blocks login with the "faol emas" message; reactivate restores it', async ({
    page,
    browser,
    baseURL
  }) => {
    const other = await newStudentContext(browser, baseURL!, 'e2e-blk')
    const { student } = other
    try {
      await other.page.goto('/')
      await expect(other.page.getByText(student.username).first()).toBeVisible()

      await searchUser(page, student.username)
      const row = userRow(page, student.username)
      await row.getByRole('button', { name: 'Bloklash' }).click()
      await expect(dialog(page).getByRole('heading', { name: 'Foydalanuvchini bloklash' })).toBeVisible()
      await dialog(page).getByRole('button', { name: 'Bloklash' }).click()
      await expect(toast(page, `${student.username} bloklandi`)).toBeVisible()
      await expect(row.getByText('Bloklangan')).toBeVisible()
      expect((await findUser(admin, student.username))?.active).toBe(false)

      // The blocked student's existing cookie is refused by the API...
      expect((await other.api.get('/api/me')).status()).toBe(403)
      // ...a reload lands on /login, and logging in again shows the "account inactive" message.
      await other.page.reload()
      await expect(other.page).toHaveURL(/\/login/)
      await uiLogin(other.page, student.username, student.password)
      await expect(other.page.getByRole('alert')).toContainText('Hisobingiz faol emas')
      await expect(other.page).toHaveURL(/\/login/)

      await row.getByRole('button', { name: 'Faollashtirish' }).click()
      await expect(dialog(page).getByRole('heading', { name: 'Foydalanuvchini faollashtirish' })).toBeVisible()
      await dialog(page).getByRole('button', { name: 'Faollashtirish' }).click()
      await expect(toast(page, `${student.username} faollashtirildi`)).toBeVisible()
      await expect(row.getByText('Faol', { exact: true })).toBeVisible()
      expect((await findUser(admin, student.username))?.active).toBe(true)

      await uiLogin(other.page, student.username, student.password)
      await expect(other.page).toHaveURL(/\/$/)
    } finally {
      await other.close()
    }
  })

  test("users: a deactivated student's open page is sent to /login on its next API call", async ({
    page,
    browser,
    baseURL
  }) => {
    const other = await newStudentContext(browser, baseURL!, 'e2e-kick')
    const { student } = other
    try {
      await other.page.goto('/')
      await expect(other.page.getByText(student.username).first()).toBeVisible()

      await searchUser(page, student.username)
      const row = userRow(page, student.username)
      await row.getByRole('button', { name: 'Bloklash' }).click()
      await dialog(page).getByRole('button', { name: 'Bloklash' }).click()
      await expect(row.getByText('Bloklangan')).toBeVisible()

      // In-app navigation triggers GET /api/progress with the now-disabled cookie.
      const progressCall = other.page.waitForResponse((r) => new URL(r.url()).pathname === '/api/progress')
      await other.page.getByRole('navigation', { name: 'Asosiy menyu' }).getByRole('link', { name: 'Natijalar' }).click()
      expect((await progressCall).status()).toBe(403)
      await expect(other.page).toHaveURL(/\/login/, { timeout: 5_000 })
    } finally {
      const u = await findUser(admin, student.username)
      if (u) await admin.put(`/api/admin/users/${u.id}`, { data: { active: true } })
      await other.close()
    }
  })

  test('users: delete removes the row and the account can no longer log in', async ({ page, baseURL }) => {
    const { student, api } = await apiStudent(baseURL!, 'e2e-del')
    await api.dispose()

    await searchUser(page, student.username)
    const row = userRow(page, student.username)
    await row.getByRole('button', { name: `${student.username} ni o'chirish` }).click()
    await expect(dialog(page).getByRole('heading', { name: "Foydalanuvchini o'chirish" })).toBeVisible()
    await dialog(page).getByRole('button', { name: "O'chirish" }).click()
    await expect(toast(page, `${student.username} o'chirildi`)).toBeVisible()
    await expect(row).toHaveCount(0)
    await expect(page.getByText('Hech narsa topilmadi')).toBeVisible()
    expect(await findUser(admin, student.username)).toBeUndefined()

    const fresh = await playwrightRequest.newContext({ baseURL, extraHTTPHeaders: apiHeaders(baseURL!) })
    const login = await fresh.post('/api/auth/login', { data: { username: student.username, password: student.password } })
    expect(login.status()).toBe(401)
    await fresh.dispose()
  })

  test("users: the admin's own row actions are disabled", async ({ page }) => {
    await page.goto('/admin?section=users')
    const self = page.getByRole('row').filter({ has: page.getByText('(siz)') })
    await expect(self).toHaveCount(1)
    await expect(self.getByRole('button', { name: /qilish$/ })).toBeDisabled()
    await expect(self.getByRole('button', { name: /^(Bloklash|Faollashtirish)$/ })).toBeDisabled()
    await expect(self.getByRole('button', { name: /ni o'chirish$/ })).toBeDisabled()
  })

  test('sessions: admin kills a running student session', async ({ page, baseURL }) => {
    const { student, api } = await apiStudent(baseURL!, 'e2e-ses')
    try {
      const sessionId = await startSession(api)
      await page.goto('/admin?section=sessions')
      const row = page.getByRole('row').filter({ has: page.getByRole('rowheader', { name: student.username, exact: true }) })
      await expect(row).toBeVisible()
      await expect(row.getByTitle(sessionId)).toBeVisible()

      await row.getByRole('button', { name: new RegExp(`^${student.username} sessiyasini to'xtatish`) }).click()
      await expect(dialog(page).getByRole('heading', { name: "Sessiyani to'xtatish" })).toBeVisible()
      await dialog(page).getByRole('button', { name: 'kill', exact: true }).click()
      await expect(toast(page, `${student.username} sessiyasi to'xtatildi`)).toBeVisible()
      await expect(row).toHaveCount(0)

      await expect.poll(async () => (await api.get('/api/sessions/active')).status()).toBe(204)
    } finally {
      await stopActiveSession(api)
      await api.dispose()
    }
  })

  test('activity: shows the student login with level colours and the level filter works', async ({ page, baseURL }) => {
    const { student, api } = await apiStudent(baseURL!, 'e2e-act')
    try {
      // One INFO "login" and one WARN "login_failed" entry for this student.
      expect((await api.post('/api/auth/login', { data: { username: student.username, password: student.password } })).status()).toBe(200)
      expect((await api.post('/api/auth/login', { data: { username: student.username, password: 'wrong-password-1' } })).status()).toBe(401)

      await page.goto('/admin?section=activity')
      const log = page.getByRole('list', { name: 'Faollik yozuvlari' })
      const loginEntry = log.getByRole('listitem').filter({ hasText: student.username }).filter({ hasText: 'User logged in' })
      const failedEntry = log.getByRole('listitem').filter({ hasText: student.username }).filter({ hasText: 'login_failed' })
      await expect(loginEntry).toHaveCount(1)
      await expect(loginEntry.getByText('login', { exact: true })).toBeVisible()
      await expect(loginEntry.getByText('INFO', { exact: true })).toHaveCSS('color', INFO_COLOR)
      await expect(failedEntry).toHaveCount(1)
      await expect(failedEntry.getByText('WARN', { exact: true })).toHaveCSS('color', WARN_COLOR)

      const filter = page.getByRole('group', { name: "Daraja bo'yicha filtr" })
      await filter.getByRole('button', { name: /^WARN/ }).click()
      await expect(filter.getByRole('button', { name: /^WARN/ })).toHaveAttribute('aria-pressed', 'true')
      await expect(failedEntry).toHaveCount(1)
      await expect(loginEntry).toHaveCount(0)
      const levels = await log.locator('li > span:first-of-type').allTextContents()
      expect(levels.length).toBeGreaterThan(0)
      expect(new Set(levels.map((l) => l.trim()))).toEqual(new Set(['WARN']))

      await filter.getByRole('button', { name: /^INFO/ }).click()
      await expect(loginEntry).toHaveCount(1)
      await expect(failedEntry).toHaveCount(0)

      await filter.getByRole('button', { name: /^Barchasi/ }).click()
      await expect(loginEntry).toHaveCount(1)
      await expect(failedEntry).toHaveCount(1)
    } finally {
      await api.dispose()
    }
  })

  test('resources: the containers meter counts a running session', async ({ page, baseURL }) => {
    const { student, api } = await apiStudent(baseURL!, 'e2e-res')
    try {
      await startSession(api)
      await page.goto('/admin?section=resources')
      const value = page.locator('main span').filter({ hasText: /^containers$/ }).locator('xpath=following-sibling::span[1]')
      await expect(value).toHaveText(/^\d+\/\d+$/)
      const [running, limit] = (await value.textContent())!.split('/').map(Number)
      expect(running).toBeGreaterThanOrEqual(1)
      expect(limit).toBeGreaterThanOrEqual(running)
      const meter = page.getByRole('meter', { name: 'Ishlayotgan konteynerlar limitga nisbatan' })
      expect(Number(await meter.getAttribute('aria-valuenow'))).toBeGreaterThan(0)
      await expect(page.getByRole('rowheader', { name: new RegExp(`^${student.username}`) })).toBeVisible()
    } finally {
      await stopActiveSession(api)
      await api.dispose()
    }
  })

  test('tasks: "Qayta yuklash" reloads all six tasks', async ({ page }) => {
    await page.goto('/admin?section=tasks')
    await page.getByRole('button', { name: 'Qayta yuklash' }).click()
    await expect(toast(page, `${TASK_COUNT} ta vazifa qayta yuklandi`)).toBeVisible()
    await expect(page.getByRole('button', { name: `${TASK_COUNT} ta yuklandi` })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Vazifalar katalogi' }).getByRole('listitem')).toHaveCount(TASK_COUNT)
  })
})

test.describe('admin access control', () => {
  test('a student cannot open /admin and the admin API answers 403', async ({ page, student, studentApi }) => {
    void student
    await page.goto('/admin')
    await expect(page).toHaveURL(/\/$/)
    await expect(page.getByRole('heading', { level: 1, name: /^Salom,/ })).toBeVisible()
    await expect(page.getByRole('link', { name: 'admin', exact: true })).toHaveCount(0)
    expect((await studentApi.get('/api/admin/stats')).status()).toBe(403)
    expect((await studentApi.get('/api/admin/users')).status()).toBe(403)
  })
})
