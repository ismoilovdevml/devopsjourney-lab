/**
 * Every page, as a student and as the admin: no console errors, no 5xx responses, no horizontal
 * scroll. With CAPTURE_SCREENSHOTS=1 it also writes the README screenshots to screenshots/desktop/.
 */
import type { Page } from '@playwright/test'
import { test, expect, newStudent, registerStudent, stopActiveSession } from '../fixtures'
import { ADMIN_STATE } from '../fixtures/env'
import {
  FIRST_TASK,
  capture,
  expectClean,
  expectNoErrorState,
  expectNoHorizontalScroll,
  openLab,
  runCheck,
  runInTerminal,
  terminalRows,
  typeSolution,
  watchPage,
  type PageProblems
} from '../fixtures/admin'

interface PageCheck {
  name: string
  path: string
  /** Waits for the page's real content (not a skeleton). */
  ready: (page: Page) => Promise<void>
}

async function settle(page: Page): Promise<void> {
  await page.waitForLoadState('networkidle')
  await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
}

async function verify(page: Page, problems: PageProblems, where: string): Promise<void> {
  await settle(page)
  await expectNoErrorState(page, where)
  await expectNoHorizontalScroll(page, where)
  expectClean(problems, where)
  problems.consoleErrors.length = 0
  problems.serverErrors.length = 0
}

const heading = (page: Page, name: string | RegExp) => page.getByRole('heading', { level: 1, name })

test.describe('pages · desktop', () => {
  test('login page (logged out)', async ({ page }) => {
    const problems = watchPage(page, { allowUnauthenticatedMe: true })
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Qaytganingizdan xursandmiz' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Kirish', exact: true })).toBeVisible()
    await verify(page, problems, 'login')
    await capture(page, 'desktop', 'login')

    // A protected URL bounces to /login with a redirect back.
    await page.goto('/progress')
    await expect(page).toHaveURL(/\/login\?redirect=(%2F|\/)progress$/)
    await verify(page, problems, 'login redirect')

    await page.goto('/no-such-page')
    await expect(heading(page, '404 · Sahifa topilmadi')).toBeVisible()
    await verify(page, problems, '404 (logged out)')
  })

  test('student: solve task 1 in the terminal, then every page is clean', async ({ page }) => {
    test.setTimeout(300_000)
    const student = newStudent('sardor')
    await registerStudent(page.request, student)
    const problems = watchPage(page)

    try {
      // Lab: start the container, type the reference solution, inspect nginx and check.
      await openLab(page, FIRST_TASK)
      await typeSolution(page, FIRST_TASK)
      await runInTerminal(page, 'clear')
      await runInTerminal(page, 'ls -la /etc/nginx', 'nginx.conf')
      await expect(terminalRows(page)).toContainText('sites-enabled')
      const score = await runCheck(page)
      const [passed, total] = score.replace(' passed', '').split('/').map(Number)
      expect(total).toBeGreaterThan(0)
      expect(passed, `check result ${score}`).toBe(total)
      await expect(page.getByRole('tabpanel').getByRole('img', { name: "O'tdi" })).toHaveCount(total)
      await expect(page.getByRole('button', { name: /Keyingi vazifa/ })).toBeVisible()
      await verify(page, problems, 'lab (active)')
      await capture(page, 'desktop', 'lab')

      const pages: PageCheck[] = [
        {
          name: 'dashboard',
          path: '/',
          ready: async (p) => {
            await expect(heading(p, /^Salom,/)).toBeVisible()
            await expect(p.getByText(/^1\s*\/\s*6/).first()).toBeVisible()
          }
        },
        {
          name: 'tasks',
          path: '/tasks?task=docker-basics',
          ready: async (p) => {
            await expect(heading(p, 'Vazifalar')).toBeVisible()
            const selected = p.getByRole('region', { name: "Vazifalar ro'yxati" }).getByRole('button', { pressed: true })
            await expect(selected).toHaveCount(1)
            await expect(selected).toContainText(/Docker/i)
            await expect(p.getByRole('button', { name: 'Laboratoriyani boshlash' })).toBeVisible()
          }
        },
        {
          name: 'progress',
          path: '/progress',
          ready: async (p) => {
            await expect(heading(p, 'Natijalar')).toBeVisible()
            await expect(p.locator('table')).toBeVisible()
          }
        },
        {
          name: 'leaderboard',
          path: '/leaderboard',
          ready: async (p) => {
            await expect(heading(p, 'Reyting')).toBeVisible()
            await expect(p.getByText(student.username).first()).toBeVisible()
          }
        },
        {
          name: 'profile',
          path: '/profile',
          ready: async (p) => {
            await expect(heading(p, student.username)).toBeVisible()
            await expect(p.getByRole('list', { name: 'Statistika' })).toContainText(/[1-9]\d* xp/)
          }
        },
        {
          name: 'profile-appearance',
          path: '/profile?tab=appearance',
          ready: async (p) => {
            await expect(p.getByRole('radiogroup', { name: 'Terminal mavzusi' })).toBeVisible()
          }
        },
        {
          name: 'profile-security',
          path: '/profile?tab=security',
          ready: async (p) => {
            await expect(p.getByRole('button', { name: 'Parolni yangilash' })).toBeVisible()
          }
        },
        {
          name: 'not-found',
          path: '/no-such-page',
          ready: async (p) => {
            await expect(heading(p, '404 · Sahifa topilmadi')).toBeVisible()
          }
        }
      ]

      for (const def of pages) {
        await test.step(def.name, async () => {
          await page.goto(def.path)
          await def.ready(page)
          await verify(page, problems, def.name)
          if (['dashboard', 'tasks', 'progress', 'leaderboard', 'profile'].includes(def.name)) {
            await capture(page, 'desktop', def.name)
          }
        })
      }

      // The student really earned the points shown above.
      const progress = await (await page.request.get('/api/progress')).json()
      expect(progress.completed).toBe(1)
      expect(progress.totalScore).toBeGreaterThan(0)
    } finally {
      await stopActiveSession(page.request)
    }
  })

  test.describe('as admin', () => {
    test.use({ storageState: ADMIN_STATE })

    test('admin console sections and student pages are clean', async ({ page }) => {
      test.setTimeout(120_000)
      const problems = watchPage(page)

      const sections: [string, string | RegExp][] = [
        ['overview', "Umumiy ko'rinish"],
        ['users', 'Foydalanuvchilar'],
        ['sessions', 'Faol sessiyalar'],
        ['activity', 'Faollik jurnali'],
        ['resources', 'Server resurslari'],
        ['tasks', 'Vazifalar']
      ]
      for (const [section, title] of sections) {
        await test.step(`admin ${section}`, async () => {
          await page.goto(`/admin?section=${section}`)
          await expect(heading(page, title)).toBeVisible()
          await expect(page.getByRole('banner').getByRole('status').filter({ hasText: 'all systems operational' })).toBeVisible()
          await verify(page, problems, `admin ${section}`)
          // The user table grows with every test run, so keep that shot to the viewport.
          if (section === 'overview') await capture(page, 'desktop', 'admin-overview')
          if (section === 'users') await capture(page, 'desktop', 'admin-users', { fullPage: false })
        })
      }

      const studentPages: [string, string, (p: Page) => Promise<void>][] = [
        ['dashboard', '/', async (p) => expect(heading(p, /^Salom,/)).toBeVisible()],
        ['tasks', '/tasks', async (p) => expect(heading(p, 'Vazifalar')).toBeVisible()],
        [
          'lab start',
          `/lab/${FIRST_TASK}`,
          async (p) => expect(p.getByRole('button', { name: 'Muhitni ishga tushirish' }).or(p.locator('.xterm'))).toBeVisible()
        ],
        ['progress', '/progress', async (p) => expect(heading(p, 'Natijalar')).toBeVisible()],
        ['leaderboard', '/leaderboard', async (p) => expect(heading(p, 'Reyting')).toBeVisible()],
        ['profile', '/profile', async (p) => expect(p.getByText(/Administrator ·/)).toBeVisible()]
      ]
      for (const [name, path, ready] of studentPages) {
        await test.step(`admin on ${name}`, async () => {
          await page.goto(path)
          await ready(page)
          if (!path.startsWith('/lab')) await expect(page.getByRole('link', { name: 'admin', exact: true })).toBeVisible()
          await verify(page, problems, `admin on ${name}`)
        })
      }
    })
  })
})
