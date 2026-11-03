/**
 * Mobile layout (390×844, "mobile" project): no horizontal page scroll, primary actions visible and
 * at least 40px tall, header navigation usable, tables inside horizontal scrollers.
 * With CAPTURE_SCREENSHOTS=1 it also writes screenshots/mobile/<page>.png.
 */
import type { Page } from '@playwright/test'
import { test, expect } from '../fixtures'
import { ADMIN_STATE } from '../fixtures/env'
import {
  FIRST_TASK,
  capture,
  expectClean,
  expectNoErrorState,
  expectNoHorizontalScroll,
  expectTablesScrollable,
  expectTappable,
  runCheck,
  waitForShell,
  watchPage,
  type PageProblems
} from '../fixtures/admin'

async function checkLayout(page: Page, problems: PageProblems, name: string): Promise<void> {
  await page.waitForLoadState('networkidle')
  await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
  await expectNoErrorState(page, name)
  await expectNoHorizontalScroll(page, name)
  await expectTablesScrollable(page, name)
  expectClean(problems, name)
}

const heading = (page: Page, name: string | RegExp) => page.getByRole('heading', { level: 1, name })

test.describe('responsive · mobile', () => {
  test('login', async ({ page }) => {
    const problems = watchPage(page, { allowUnauthenticatedMe: true })
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Qaytganingizdan xursandmiz' })).toBeVisible()
    await expectTappable(page.getByRole('button', { name: 'Kirish', exact: true }), 'login submit')
    await expectTappable(page.getByLabel('Foydalanuvchi nomi'), 'username input')
    await expectTappable(page.getByLabel('Parol', { exact: true }), 'password input')
    await checkLayout(page, problems, 'login')
    await capture(page, 'mobile', 'login')

    // Switching to the register tab keeps the layout intact.
    await page.getByRole('tab', { name: "Ro'yxatdan o'tish" }).tap()
    await expectTappable(page.getByRole('button', { name: "Ro'yxatdan o'tish", exact: true }), 'register submit')
    await expectNoHorizontalScroll(page, 'register')
  })

  test('student pages', async ({ page, student }) => {
    test.setTimeout(180_000)
    const problems = watchPage(page)

    await test.step('dashboard + header nav', async () => {
      await page.goto('/')
      await expect(heading(page, /^Salom,/)).toBeVisible()
      await expectTappable(page.getByRole('link', { name: 'Terminalni ochish' }), 'dashboard: open terminal')
      const nav = page.getByRole('navigation', { name: 'Asosiy menyu' })
      await expect(nav).toBeVisible()
      for (const label of ['Bosh sahifa', 'Vazifalar', 'Laboratoriya', 'Natijalar', 'Reyting']) {
        const link = nav.getByRole('link', { name: label })
        await link.scrollIntoViewIfNeeded()
        await expect(link, `nav link ${label}`).toBeInViewport()
      }
      // Header chrome (38px icon controls) is not a primary action: it must be visible and on screen.
      await expect(page.getByRole('link', { name: `Profil: ${student.username}` })).toBeInViewport()
      await expect(page.getByRole('button', { name: 'Chiqish' })).toBeInViewport()
      await checkLayout(page, problems, 'dashboard')
      await capture(page, 'mobile', 'dashboard')

      // The header nav actually navigates.
      await nav.getByRole('link', { name: 'Vazifalar' }).tap()
      await expect(page).toHaveURL(/\/tasks/)
      await expect(heading(page, 'Vazifalar')).toBeVisible()
    })

    await test.step('tasks', async () => {
      await page.goto(`/tasks?task=${FIRST_TASK}`)
      await expect(heading(page, 'Vazifalar')).toBeVisible()
      await expectTappable(page.getByRole('button', { name: 'Laboratoriyani boshlash' }), 'tasks: start lab')
      await checkLayout(page, problems, 'tasks')
      await page.evaluate(() => window.scrollTo(0, 0))
      await capture(page, 'mobile', 'tasks')
    })

    await test.step('lab start screen', async () => {
      await page.goto(`/lab/${FIRST_TASK}`)
      const start = page.getByRole('button', { name: 'Muhitni ishga tushirish' })
      await expectTappable(start, 'lab: start environment')
      await checkLayout(page, problems, 'lab start')
      await page.evaluate(() => window.scrollTo(0, 0))
      await capture(page, 'mobile', 'lab-start')
      await start.tap()
    })

    await test.step('lab active', async () => {
      await waitForShell(page)
      // Top-bar session controls are secondary (36px "sm" buttons) and scroll with the page on
      // mobile: they must be on screen at the top of the page.
      await page.evaluate(() => window.scrollTo(0, 0))
      await expect(page.getByRole('button', { name: 'Tugatish' })).toBeInViewport()
      await expect(page.getByRole('button', { name: 'Qayta yaratish' })).toBeInViewport()
      await expectTappable(
        page.locator('section[aria-label="Terminal"]').getByRole('button', { name: /Yechimni tekshirish/ }),
        'lab: check'
      )
      const score = await runCheck(page)
      expect(score).toMatch(/^\d+\/\d+ passed$/)
      await checkLayout(page, problems, 'lab active')
      await page.evaluate(() => window.scrollTo(0, 0))
      await capture(page, 'mobile', 'lab-active')
    })

    await test.step('progress', async () => {
      await page.goto('/progress')
      await expect(heading(page, 'Natijalar')).toBeVisible()
      await expect(page.locator('table')).toBeAttached()
      await expectTappable(page.getByRole('button', { name: 'Progressni tiklash' }), 'progress: reset')
      await checkLayout(page, problems, 'progress')
      await capture(page, 'mobile', 'progress')
    })

    await test.step('leaderboard', async () => {
      await page.goto('/leaderboard')
      await expect(heading(page, 'Reyting')).toBeVisible()
      await expect(page.locator('table')).toBeAttached()
      await expectTappable(page.getByRole('link', { name: /^(Ball yig'ish|Vazifalarga o'tish)$/ }), 'leaderboard: CTA')
      await checkLayout(page, problems, 'leaderboard')
      await capture(page, 'mobile', 'leaderboard')
    })

    const tabs: [string, string, (p: Page) => ReturnType<Page['getByRole']>][] = [
      ['profile', 'Profil', (p) => p.getByRole('button', { name: 'Saqlash' })],
      ['appearance', "Ko'rinish", (p) => p.getByRole('radio', { name: 'Matrix', exact: true })],
      ['notifications', 'Bildirishnomalar', (p) => p.getByRole('switch').first()],
      ['security', 'Xavfsizlik', (p) => p.getByRole('button', { name: 'Parolni yangilash' })]
    ]
    for (const [tab, label, primary] of tabs) {
      await test.step(`profile ${tab}`, async () => {
        await page.goto('/profile')
        const nav = page.getByRole('navigation', { name: "Sozlamalar bo'limlari" })
        const link = nav.getByRole('link', { name: label })
        await expectTappable(link, `profile tab ${label}`)
        await link.tap()
        await expect(page).toHaveURL(new RegExp(`tab=${tab}`))
        await expect(link).toHaveAttribute('aria-current', 'page')
        const action = primary(page)
        if (tab === 'notifications') {
          // Switches are "tez orada" (disabled); they only need to be visible.
          await expect(action).toBeVisible()
        } else {
          await expectTappable(action, `profile ${tab}: primary action`)
        }
        await checkLayout(page, problems, `profile ${tab}`)
        await page.evaluate(() => window.scrollTo(0, 0))
        await capture(page, 'mobile', `profile-${tab}`)
      })
    }
  })

  test.describe('admin', () => {
    test.use({ storageState: ADMIN_STATE })

    test('admin sections', async ({ page }) => {
      const problems = watchPage(page)
      const sections: [string, string, string][] = [
        ['overview', 'Umumiy', "Umumiy ko'rinish"],
        ['users', 'Foydalanuvchilar', 'Foydalanuvchilar'],
        ['sessions', 'Sessiyalar', 'Faol sessiyalar'],
        ['activity', 'Faollik jurnali', 'Faollik jurnali'],
        ['resources', 'Resurslar', 'Server resurslari'],
        ['tasks', 'Vazifalar', 'Vazifalar']
      ]
      await page.goto('/admin')
      const nav = page.getByRole('navigation', { name: "Admin bo'limlari" })
      for (const [section, label, title] of sections) {
        await test.step(section, async () => {
          const link = nav.getByRole('link', { name: new RegExp(`^${label}`) })
          await expectTappable(link, `admin tab ${label}`)
          await link.tap()
          await expect(heading(page, title)).toBeVisible()
          await expect(link).toHaveAttribute('aria-current', 'page')
          await expect(link).toBeInViewport()
          await expect(page.getByRole('link', { name: 'Ilova' })).toBeInViewport()
          await expect(page.getByText('Tizim holati: all systems operational')).toBeAttached()
          if (section === 'users') await expect(page.locator('table')).toBeAttached()
          if (section === 'tasks') await expectTappable(page.getByRole('button', { name: 'Qayta yuklash' }), 'admin tasks: reload')
          await checkLayout(page, problems, `admin ${section}`)
          await page.evaluate(() => window.scrollTo(0, 0))
          await capture(page, 'mobile', `admin-${section}`, { fullPage: section !== 'users' })
        })
      }
    })
  })
})
