import { test, expect } from '../fixtures'
import { headerProfileLink, pageTitle } from '../fixtures/auth'
import type { Page } from '@playwright/test'

const NAV_ITEMS = [
  { label: 'Bosh sahifa', path: '/', title: 'Bosh sahifa' },
  { label: 'Vazifalar', path: '/tasks', title: 'Vazifalar' },
  { label: 'Natijalar', path: '/progress', title: 'Natijalar' },
  { label: 'Reyting', path: '/leaderboard', title: 'Reyting' }
] as const

function mainNav(page: Page) {
  return page.getByRole('navigation', { name: 'Asosiy menyu' })
}

function pathOf(page: Page): string {
  return new URL(page.url()).pathname
}

async function expectActiveNav(page: Page, label: string): Promise<void> {
  const links = mainNav(page).getByRole('link')
  await expect(mainNav(page).getByRole('link', { name: label, exact: true })).toHaveAttribute('aria-current', 'page')
  // Exactly one item is current.
  await expect(mainNav(page).locator('[aria-current="page"]')).toHaveCount(1)
  await expect(links).toHaveCount(5)
}

test('header nav reaches every page with aria-current and a page title', async ({ page, student }) => {
  await page.goto('/')
  await expect(headerProfileLink(page, student.username)).toBeVisible()

  // Visit in an order where every click changes the page.
  for (const item of [...NAV_ITEMS.slice(1), NAV_ITEMS[0]]) {
    await mainNav(page).getByRole('link', { name: item.label, exact: true }).click()
    await expect.poll(() => pathOf(page)).toBe(item.path)
    await expect(page).toHaveTitle(pageTitle(item.title))
    await expectActiveNav(page, item.label)
    await expect(page.locator('main#main')).toBeVisible()
  }
})

test('"Laboratoriya" without a running session falls back to the task list', async ({ page, student }) => {
  await page.goto('/progress')
  await expect(headerProfileLink(page, student.username)).toBeVisible()

  const lab = mainNav(page).getByRole('link', { name: 'Laboratoriya', exact: true })
  await expect(lab).toHaveAttribute('href', '/lab')
  await lab.click()

  // LabView redirects to /tasks when there is no active session to resume.
  await expect.poll(() => pathOf(page)).toBe('/tasks')
  await expect(page).toHaveTitle(pageTitle('Vazifalar'))
  await expectActiveNav(page, 'Vazifalar')
})

test('lab page for a task has its own chrome and title', async ({ page, studentApi }) => {
  const res = await studentApi.get('/api/tasks')
  const [first] = (await res.json()) as { id: string; title: string }[]

  await page.goto(`/lab/${first.id}`)
  await expect(page).toHaveTitle(`${first.title} · Laboratoriya · ${'DevOps Journey Lab'}`)
  await expect(page.getByRole('navigation', { name: "Yo'l" }).getByText(first.id)).toHaveAttribute(
    'aria-current',
    'page'
  )
  await expect(page.getByRole('button', { name: 'Muhitni ishga tushirish' })).toBeVisible()
  // The lab replaces the main nav with its own top bar.
  await expect(mainNav(page)).toHaveCount(0)
})

test('profile avatar link opens /profile with aria-current', async ({ page, student }) => {
  await page.goto('/')
  await headerProfileLink(page, student.username).click()
  await expect.poll(() => pathOf(page)).toBe('/profile')
  await expect(page).toHaveTitle(pageTitle('Profil'))
  await expect(headerProfileLink(page, student.username)).toHaveAttribute('aria-current', 'page')
  await expect(mainNav(page).locator('[aria-current="page"]')).toHaveCount(0)
})

test('Ctrl/Cmd+K opens /tasks with the search box focused', async ({ page, student }) => {
  await page.goto('/leaderboard')
  await expect(headerProfileLink(page, student.username)).toBeVisible()

  await page.keyboard.press('ControlOrMeta+k')
  await expect.poll(() => pathOf(page)).toBe('/tasks')
  const search = page.getByRole('searchbox', { name: 'Vazifa qidirish' })
  await expect(search).toBeFocused()
  // The one-shot ?focus=search flag is dropped from the URL.
  await expect.poll(() => new URL(page.url()).searchParams.has('focus')).toBe(false)

  // The header search button does the same.
  await page.goto('/progress')
  await page.getByRole('button', { name: 'Qidirish' }).click()
  await expect.poll(() => pathOf(page)).toBe('/tasks')
  await expect(search).toBeFocused()
})

test.describe('404', () => {
  test('unknown route shows the not-found page when logged out', async ({ page }) => {
    const res = await page.goto('/definitely/not-a-page')
    expect(res?.status()).toBe(200) // SPA fallback; the router renders the 404 view.
    await expect(page.getByRole('heading', { level: 1, name: '404 · Sahifa topilmadi' })).toBeVisible()
    await expect(page).toHaveTitle(pageTitle('Topilmadi'))
    await expect(page.getByText('/definitely/not-a-page: No such file or directory')).toBeVisible()

    await page.getByRole('link', { name: 'Kirish sahifasiga' }).click()
    await expect.poll(() => pathOf(page)).toBe('/login')
  })

  test('unknown route offers the way home when logged in', async ({ page, student }) => {
    await page.goto('/no-such-route')
    await expect(page.getByRole('heading', { level: 1, name: '404 · Sahifa topilmadi' })).toBeVisible()
    await expect(page.getByText(`${student.username}@lab`).first()).toBeVisible()

    await page.getByRole('link', { name: 'Vazifalar' }).click()
    await expect.poll(() => pathOf(page)).toBe('/tasks')

    await page.goto('/still/missing')
    await page.getByRole('link', { name: 'Bosh sahifaga' }).click()
    await expect.poll(() => pathOf(page)).toBe('/')
  })
})

test.describe('admin area', () => {
  test('student cannot open /admin and does not see the admin link', async ({ page, student }) => {
    await page.goto('/admin')
    await expect.poll(() => pathOf(page)).toBe('/')
    await expect(headerProfileLink(page, student.username)).toBeVisible()
    await expect(page.getByRole('banner').getByRole('link', { name: 'admin', exact: true })).toHaveCount(0)

    // The API agrees.
    expect((await page.request.get('/api/admin/stats')).status()).toBe(403)
  })

  test('admin sees the admin link and can open /admin', async ({ adminPage }) => {
    await adminPage.goto('/')
    const adminLink = adminPage.getByRole('banner').getByRole('link', { name: 'admin', exact: true })
    await expect(adminLink).toBeVisible()
    await adminLink.click()
    await expect.poll(() => pathOf(adminPage)).toBe('/admin')
    await expect(adminPage).toHaveTitle(pageTitle('Admin'))
  })
})

test.describe('history with query-synced filters', () => {
  test('/tasks filters survive back/forward', async ({ page, student }) => {
    void student
    await page.goto('/tasks')
    const level = page.getByRole('group', { name: "Daraja bo'yicha" })
    const status = page.getByRole('group', { name: "Holat bo'yicha" })
    const search = page.getByRole('searchbox', { name: 'Vazifa qidirish' })
    await expect(level.getByRole('button', { name: 'Har qanday' })).toHaveAttribute('aria-pressed', 'true')

    await level.getByRole('button', { name: 'Oson' }).click()
    await status.getByRole('button', { name: 'Yangi' }).click()
    await search.fill('nginx')
    await expect
      .poll(() => Object.fromEntries(new URL(page.url()).searchParams))
      .toMatchObject({ level: 'easy', status: 'new', q: 'nginx' })

    await mainNav(page).getByRole('link', { name: 'Reyting', exact: true }).click()
    await expect.poll(() => pathOf(page)).toBe('/leaderboard')

    await page.goBack()
    await expect.poll(() => pathOf(page)).toBe('/tasks')
    await expect(level.getByRole('button', { name: 'Oson' })).toHaveAttribute('aria-pressed', 'true')
    await expect(status.getByRole('button', { name: 'Yangi' })).toHaveAttribute('aria-pressed', 'true')
    await expect(search).toHaveValue('nginx')
    await expect(page.getByRole('status').filter({ hasText: 'ta vazifa topildi' })).toHaveText(/^[1-9]\d* ta vazifa topildi$/)

    await page.goForward()
    await expect.poll(() => pathOf(page)).toBe('/leaderboard')
    await expectActiveNav(page, 'Reyting')

    await page.goBack()
    await expect(search).toHaveValue('nginx')

    // Filters are shareable: a fresh load of the URL restores them too.
    await page.reload()
    await expect(level.getByRole('button', { name: 'Oson' })).toHaveAttribute('aria-pressed', 'true')
    await expect(search).toHaveValue('nginx')
  })

  test('/tasks search with no match shows the empty state and can be reset', async ({ page, student }) => {
    void student
    await page.goto('/tasks?q=zzz-no-such-task-zzz')
    await expect(page.getByRole('heading', { name: 'Hech narsa topilmadi' })).toBeVisible()
    await page.getByRole('button', { name: 'Filtrlarni tozalash' }).click()
    await expect.poll(() => new URL(page.url()).search).toBe('')
    await expect(page.getByRole('searchbox', { name: 'Vazifa qidirish' })).toHaveValue('')
  })

  test('/leaderboard?period= is synced with the period control across back/forward', async ({ page, student }) => {
    void student
    const period = page.getByRole('group', { name: 'Davr' })

    const monthRequest = page.waitForRequest((r) => r.url().includes('/api/leaderboard') && r.url().includes('period=month'))
    await page.goto('/leaderboard?period=month')
    await monthRequest
    await expect(period.getByRole('button', { name: 'Bu oy' })).toHaveAttribute('aria-pressed', 'true')

    const weekRequest = page.waitForRequest((r) => r.url().includes('/api/leaderboard') && r.url().includes('period=week'))
    await period.getByRole('button', { name: 'Bu hafta' }).click()
    await weekRequest
    await expect.poll(() => new URL(page.url()).searchParams.get('period')).toBe('week')
    await expect(period.getByRole('button', { name: 'Bu hafta' })).toHaveAttribute('aria-pressed', 'true')
    await expect(period.getByRole('button', { name: 'Bu oy' })).toHaveAttribute('aria-pressed', 'false')

    await mainNav(page).getByRole('link', { name: 'Natijalar', exact: true }).click()
    await expect.poll(() => pathOf(page)).toBe('/progress')

    await page.goBack()
    await expect.poll(() => new URL(page.url()).searchParams.get('period')).toBe('week')
    await expect(period.getByRole('button', { name: 'Bu hafta' })).toHaveAttribute('aria-pressed', 'true')

    await page.goForward()
    await expect.poll(() => pathOf(page)).toBe('/progress')
    await expectActiveNav(page, 'Natijalar')

    // Unknown period values fall back to "all".
    await page.goto('/leaderboard?period=decade')
    await expect(period.getByRole('button', { name: 'Barcha vaqt' })).toHaveAttribute('aria-pressed', 'true')
  })
})
