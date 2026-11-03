import AxeBuilder from '@axe-core/playwright'
import type { Page, TestInfo } from '@playwright/test'
import { test, expect } from '../fixtures'
import { firstTaskId } from '../fixtures/auth'

const BLOCKING = new Set(['serious', 'critical'])
/** Tags that gate the suite. WCAG 2.1 rules also run but are only reported (annotations). */
const GATING_TAGS = ['wcag2a', 'wcag2aa']
const REPORT_TAGS = [...GATING_TAGS, 'wcag21a', 'wcag21aa']

/** Wait until the page has rendered real content (no skeletons, no pending requests). */
async function settle(page: Page): Promise<void> {
  await page.waitForLoadState('networkidle')
  await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
  await expect(page.locator('main').first()).toBeVisible()
}

/**
 * Runs axe on the current page. Every violation is attached to the report and annotated;
 * serious/critical violations of WCAG 2.0 A/AA rules fail the test.
 */
async function checkA11y(page: Page, testInfo: TestInfo, label: string): Promise<void> {
  await settle(page)
  const results = await new AxeBuilder({ page }).withTags(REPORT_TAGS).analyze()

  const summary = results.violations.map((v) => ({
    id: v.id,
    impact: v.impact,
    gating: v.tags.some((t) => GATING_TAGS.includes(t)),
    help: v.help,
    nodes: v.nodes.map((n) => ({ target: n.target.join(' '), summary: n.failureSummary }))
  }))
  if (summary.length) {
    await testInfo.attach(`axe-${label}.json`, { body: JSON.stringify(summary, null, 2), contentType: 'application/json' })
    for (const v of summary) {
      testInfo.annotations.push({
        type: `axe:${v.impact}${v.gating ? '' : ' (wcag21, non-gating)'}`,
        description: `${label}: ${v.id} — ${v.nodes.map((n) => n.target).join(', ')}`
      })
    }
  }

  const blocking = summary.filter((v) => v.gating && v.impact && BLOCKING.has(v.impact))
  expect(blocking, `serious/critical axe violations on ${label}`).toEqual([])
}

test.beforeEach(async ({ page }) => {
  // Keep transitions from producing transient colour-contrast results.
  await page.emulateMedia({ reducedMotion: 'reduce' })
})

test.describe('axe (wcag2a + wcag2aa)', () => {
  test('login page', async ({ page }, testInfo) => {
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Qaytganingizdan xursandmiz' })).toBeVisible()
    await checkA11y(page, testInfo, 'login')
  })

  test('login page — register tab with validation errors', async ({ page }, testInfo) => {
    await page.goto('/login?mode=register')
    await page.getByLabel('Foydalanuvchi nomi', { exact: true }).fill('x y')
    await page.getByRole('button', { name: "Ro'yxatdan o'tish", exact: true }).click()
    await expect(page.getByLabel('Foydalanuvchi nomi', { exact: true })).toHaveAttribute('aria-invalid', 'true')
    await checkA11y(page, testInfo, 'login-register-errors')
  })

  test('dashboard', async ({ page, student }, testInfo) => {
    await page.goto('/')
    await expect(page.getByText(`${student.username}@lab`)).toBeVisible()
    await checkA11y(page, testInfo, 'dashboard')
  })

  test('tasks', async ({ page, student }, testInfo) => {
    void student
    await page.goto('/tasks')
    await expect(page.getByRole('status').filter({ hasText: 'ta vazifa topildi' })).toBeAttached()
    await checkA11y(page, testInfo, 'tasks')
  })

  test('progress', async ({ page, student }, testInfo) => {
    void student
    await page.goto('/progress')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await checkA11y(page, testInfo, 'progress')
  })

  test('leaderboard', async ({ page, student }, testInfo) => {
    void student
    await page.goto('/leaderboard')
    await expect(page.getByRole('heading', { level: 1, name: 'Reyting' })).toBeVisible()
    await checkA11y(page, testInfo, 'leaderboard')
  })

  for (const tab of [
    { id: 'profile', label: 'Profil' },
    { id: 'appearance', label: "Ko'rinish" },
    { id: 'notifications', label: 'Bildirishnomalar' },
    { id: 'security', label: 'Xavfsizlik' }
  ]) {
    test(`profile — ${tab.id} tab`, async ({ page, student }, testInfo) => {
      void student
      await page.goto(`/profile?tab=${tab.id}`)
      await expect(
        page.getByRole('navigation', { name: "Sozlamalar bo'limlari" }).getByRole('link', { name: tab.label })
      ).toHaveAttribute('aria-current', 'page')
      await checkA11y(page, testInfo, `profile-${tab.id}`)
    })
  }

  test('lab start screen', async ({ page, studentApi }, testInfo) => {
    const taskId = await firstTaskId(studentApi)
    await page.goto(`/lab/${taskId}`)
    await expect(page.getByRole('button', { name: 'Muhitni ishga tushirish' })).toBeVisible()
    await checkA11y(page, testInfo, 'lab-start')
  })

  test('404 page', async ({ page }, testInfo) => {
    await page.goto('/no/such/page')
    await expect(page.getByRole('heading', { name: '404 · Sahifa topilmadi' })).toBeVisible()
    await checkA11y(page, testInfo, '404')
  })
})

test.describe('keyboard', () => {
  test('first Tab reaches the skip link, which moves focus to #main', async ({ page, student }) => {
    void student
    await page.goto('/tasks')
    await settle(page)

    await page.keyboard.press('Tab')
    const skip = page.getByRole('link', { name: "Asosiy kontentga o'tish" })
    await expect(skip).toBeFocused()
    await expect(skip).toBeVisible() // sr-only until focused

    await page.keyboard.press('Enter')
    await expect.poll(() => page.evaluate(() => document.activeElement?.id)).toBe('main')

    // The next Tab continues inside the main content, not back in the header.
    await page.keyboard.press('Tab')
    expect(await page.evaluate(() => !!document.activeElement?.closest('main#main'))).toBe(true)
  })

  test('lab page has its own skip link to the workspace', async ({ page, studentApi }) => {
    const taskId = await firstTaskId(studentApi)
    await page.goto(`/lab/${taskId}`)
    await expect(page.getByRole('button', { name: 'Muhitni ishga tushirish' })).toBeVisible()

    await page.keyboard.press('Tab')
    await expect(page.getByRole('link', { name: "Asosiy qismga o'tish" })).toBeFocused()
  })

  test('keyboard focus ring is visible on buttons and links', async ({ page }) => {
    await page.goto('/login')
    await settle(page)

    const seen: string[] = []
    for (let i = 0; i < 12; i++) {
      await page.keyboard.press('Tab')
      const info = await page.evaluate(() => {
        const el = document.activeElement as HTMLElement | null
        if (!el || el === document.body) return null
        const cs = getComputedStyle(el)
        return {
          tag: el.tagName.toLowerCase(),
          name: (el.getAttribute('aria-label') ?? el.textContent ?? '').trim().slice(0, 40),
          focusVisible: el.matches(':focus-visible'),
          outlineStyle: cs.outlineStyle,
          outlineWidth: parseFloat(cs.outlineWidth),
          boxShadow: cs.boxShadow
        }
      })
      if (!info || !['button', 'a'].includes(info.tag)) continue
      seen.push(`${info.tag}:${info.name}`)
      expect(info.focusVisible, `${info.tag} "${info.name}" should match :focus-visible`).toBe(true)
      const hasOutline = info.outlineStyle !== 'none' && info.outlineWidth >= 1
      expect(hasOutline || info.boxShadow !== 'none', `${info.tag} "${info.name}" has no visible focus ring`).toBe(true)
    }
    // Sanity: the walk actually covered several controls (tabs, submit, toggle…).
    expect(seen.length, seen.join(', ')).toBeGreaterThanOrEqual(3)
  })

  test('auth tabs follow the ARIA tabs pattern (arrow keys)', async ({ page }) => {
    await page.goto('/login')
    const loginTab = page.getByRole('tab', { name: 'Kirish', exact: true })
    const registerTab = page.getByRole('tab', { name: "Ro'yxatdan o'tish", exact: true })

    await loginTab.focus()
    await page.keyboard.press('ArrowRight')
    await expect(registerTab).toBeFocused()
    await expect(registerTab).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('tabpanel')).toHaveAttribute('aria-labelledby', 'auth-tab-register')

    await page.keyboard.press('Home')
    await expect(loginTab).toBeFocused()
    await expect(loginTab).toHaveAttribute('aria-selected', 'true')
  })
})
