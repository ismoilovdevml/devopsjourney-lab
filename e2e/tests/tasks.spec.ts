/**
 * /tasks for a brand-new student: grid, lock state, URL-synced filters and the detail panel.
 * No lab containers are started here.
 */
import { test, expect } from '../fixtures'
import type { Locator, Page } from '@playwright/test'
import { DOCKER, NGINX } from '../fixtures/lab'

interface TaskSummary {
  id: string
  title: string
  difficulty: number
  unlocked: boolean
  completed: boolean
  checks_count: number
}

function taskList(page: Page): Locator {
  return page.getByRole('region', { name: "Vazifalar ro'yxati" })
}

/** Task cards only (the empty state's "Filtrlarni tozalash" button is not a card). */
function cards(page: Page): Locator {
  return taskList(page).getByRole('listitem').getByRole('button')
}

function card(page: Page, title: string): Locator {
  return cards(page).filter({ hasText: new RegExp(title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')) })
}

test.describe('tasks page', () => {
  test('shows 6 task cards and only task 1 is unlocked for a new student', async ({ page, studentApi }) => {
    const tasks = (await (await studentApi.get('/api/tasks')).json()) as TaskSummary[]
    expect(tasks).toHaveLength(6)
    expect(tasks.filter((t) => t.unlocked).map((t) => t.id)).toEqual([NGINX.id])

    await page.goto('/tasks')
    await expect(cards(page)).toHaveCount(6)
    await expect(page.getByRole('status').filter({ hasText: 'ta vazifa topildi' })).toHaveText('6 ta vazifa topildi')

    for (const t of tasks) {
      const c = card(page, t.title)
      await expect(c).toBeVisible()
      if (t.id === NGINX.id) {
        await expect(c).toContainText('Yangi')
        await expect(c).not.toContainText('Qulflangan')
      } else {
        await expect(c).toContainText('Qulflangan')
      }
    }
    await expect(page.getByText('0/6 done')).toBeVisible()

    // The next open task is selected by default.
    await expect(card(page, NGINX.title)).toHaveAttribute('aria-pressed', 'true')
    await expect(page.getByRole('complementary', { name: NGINX.title })).toBeVisible()
  })

  test('search, status and level filters narrow the grid and sync to the URL', async ({ page, student }) => {
    void student
    await page.goto('/tasks')
    await expect(cards(page)).toHaveCount(6)
    const search = page.getByRole('searchbox', { name: 'Vazifa qidirish' })
    const status = page.getByRole('group', { name: "Holat bo'yicha" })
    const level = page.getByRole('group', { name: "Daraja bo'yicha" })

    // Search (debounced into ?q=).
    await search.fill('docker')
    await expect(cards(page)).toHaveCount(1)
    await expect(card(page, DOCKER.title)).toBeVisible()
    await expect(page).toHaveURL(/[?&]q=docker(&|$)/)

    // Search matches the Uzbek category label too.
    await search.fill('tarmoq')
    await expect(cards(page)).toHaveCount(1)
    await expect(card(page, 'UFW Firewall Configuration')).toBeVisible()
    await expect(page).toHaveURL(/[?&]q=tarmoq(&|$)/)

    await search.fill('')
    await expect(cards(page)).toHaveCount(6)
    await expect(page).not.toHaveURL(/[?&]q=/)

    // Level.
    await level.getByRole('button', { name: 'Qiyin' }).click()
    await expect(page).toHaveURL(/[?&]level=hard(&|$)/)
    await expect(cards(page)).toHaveCount(1)
    await expect(card(page, 'Cron Job & Backup Automation')).toBeVisible()
    await expect(level.getByRole('button', { name: 'Qiyin' })).toHaveAttribute('aria-pressed', 'true')

    await level.getByRole('button', { name: 'Oson' }).click()
    await expect(page).toHaveURL(/[?&]level=easy(&|$)/)
    await expect(cards(page)).toHaveCount(2)
    await expect(card(page, NGINX.title)).toBeVisible()
    await expect(card(page, 'Linux User & Permission Management')).toBeVisible()

    await level.getByRole('button', { name: "O'rta" }).click()
    await expect(page).toHaveURL(/[?&]level=medium(&|$)/)
    await expect(cards(page)).toHaveCount(3)

    // Status: nothing is done yet; "Yangi" also lists locked tasks.
    await level.getByRole('button', { name: 'Har qanday' }).click()
    await expect(page).not.toHaveURL(/[?&]level=/)
    await status.getByRole('button', { name: 'Yakunlangan' }).click()
    await expect(page).toHaveURL(/[?&]status=done(&|$)/)
    await expect(cards(page)).toHaveCount(0)
    await expect(page.getByText('Hech narsa topilmadi')).toBeVisible()

    await status.getByRole('button', { name: 'Jarayonda' }).click()
    await expect(page).toHaveURL(/[?&]status=active(&|$)/)
    await expect(cards(page)).toHaveCount(0)

    await status.getByRole('button', { name: 'Yangi' }).click()
    await expect(page).toHaveURL(/[?&]status=new(&|$)/)
    await expect(cards(page)).toHaveCount(6)

    // Combined filters survive a reload (URL is the source of truth).
    await level.getByRole('button', { name: 'Oson' }).click()
    await search.fill('nginx')
    await expect(page).toHaveURL(/q=nginx/)
    await expect(cards(page)).toHaveCount(1)
    await page.reload()
    await expect(cards(page)).toHaveCount(1)
    await expect(card(page, NGINX.title)).toBeVisible()
    await expect(search).toHaveValue('nginx')
    await expect(status.getByRole('button', { name: 'Yangi' })).toHaveAttribute('aria-pressed', 'true')
    await expect(level.getByRole('button', { name: 'Oson' })).toHaveAttribute('aria-pressed', 'true')

    // Empty result -> "Filtrlarni tozalash" resets everything.
    await search.fill('no-such-task-xyz')
    await expect(page.getByText('Hech narsa topilmadi')).toBeVisible()
    await page.getByRole('button', { name: 'Filtrlarni tozalash' }).click()
    await expect(cards(page)).toHaveCount(6)
    await expect(search).toHaveValue('')
    await expect(page).not.toHaveURL(/[?&](q|status|level)=/)
  })

  test('selecting a card opens the detail panel with checks and a rendered README', async ({ page, studentApi }) => {
    const detail = (await (await studentApi.get(`/api/tasks/${DOCKER.id}`)).json()) as {
      checks: { name: string }[]
      readme: string
    }
    expect(detail.checks.length).toBeGreaterThan(0)
    expect(detail.readme).toMatch(/^## /m)

    await page.goto('/tasks')
    await card(page, DOCKER.title).click()
    await expect(page).toHaveURL(new RegExp(`[?&]task=${DOCKER.id}(&|$)`))
    await expect(card(page, DOCKER.title)).toHaveAttribute('aria-pressed', 'true')

    const panel = page.getByRole('complementary', { name: DOCKER.title })
    await expect(panel).toBeVisible()
    await expect(panel).toContainText(`tasks/${DOCKER.id}`)

    // "# nima tekshiriladi" lists every check, and the "Tekshiruv" stat agrees.
    const checks = panel.getByRole('region', { name: '# nima tekshiriladi' }).getByRole('listitem')
    await expect(checks).toHaveCount(detail.checks.length)
    for (const [i, c] of detail.checks.entries()) await expect(checks.nth(i)).toContainText(c.name)
    await expect(panel.getByRole('definition').nth(1)).toHaveText(String(detail.checks.length))

    // README is collapsed, then rendered as HTML (headings, code), never raw markdown.
    const toggle = panel.getByRole('button', { name: /README\.md/ })
    await expect(toggle).toHaveAttribute('aria-expanded', 'false')
    await toggle.click()
    await expect(toggle).toHaveAttribute('aria-expanded', 'true')
    const readme = panel.getByRole('region', { name: /README\.md/ })
    await expect(readme).toBeVisible()
    await expect(readme.locator('h2').first()).toBeVisible()
    await expect(readme.locator('pre code, code').first()).toBeVisible()
    const text = (await readme.innerText()) ?? ''
    expect(text).not.toMatch(/^\s*#{1,6}\s/m)
    expect(text).not.toContain('```')

    // Locked task: notice, no start button.
    await expect(panel.getByText('Ochish uchun avvalgi vazifalarni yakunlang')).toBeVisible()
    await expect(panel.getByRole('button', { name: /Laboratoriyani boshlash|Terminalga qaytish|Qayta ishlash/ })).toHaveCount(0)
  })

  test('unlocked task 1 offers "Laboratoriyani boshlash" and no locked notice', async ({ page, student }) => {
    void student
    await page.goto(`/tasks?task=${NGINX.id}`)
    const panel = page.getByRole('complementary', { name: NGINX.title })
    await expect(panel.getByRole('region', { name: '# nima tekshiriladi' }).getByRole('listitem')).toHaveCount(NGINX.checks)
    await expect(panel.getByRole('button', { name: 'Laboratoriyani boshlash' })).toBeEnabled()
    await expect(panel.getByText('Ochish uchun avvalgi vazifalarni yakunlang')).toHaveCount(0)
  })

  test('a locked lab URL shows the locked state instead of a start screen', async ({ page, student }) => {
    void student
    await page.goto(`/lab/${DOCKER.id}`)
    await expect(page.getByRole('heading', { name: 'Bu vazifa hali qulflangan' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Muhitni ishga tushirish' })).toHaveCount(0)
    await page.getByRole('link', { name: 'Vazifalarga qaytish' }).click()
    await expect(page).toHaveURL(/\/tasks$/)
  })
})
