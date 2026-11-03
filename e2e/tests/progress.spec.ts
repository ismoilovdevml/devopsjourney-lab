/**
 * /progress after completing task 1 through the real lab UI, then the "Progressni tiklash" danger flow.
 */
import { test, expect } from '../fixtures'
import { DOCKER, NGINX, completeNginxViaUi, dialogButton } from '../fixtures/lab'

test.describe('progress page', () => {
  test('shows the completed task, web skill, first achievement, and resets back to zero', async ({
    page,
    student,
    studentApi
  }) => {
    test.setTimeout(180_000)
    await completeNginxViaUi(page)

    await page.goto('/progress')
    const overview = page.getByRole('region', { name: "Umumiy ko'rsatkichlar" })
    await expect(overview).toContainText(/Yakunlangan\s*1\/6/)
    await expect(overview).toContainText(/Urinishlar\s*1(?!\d)/)
    await expect(overview).toContainText(/(?<!\d)100 \/ \d+ xp/)

    // Per-task table row.
    const table = page.getByRole('table', { name: "Har bir vazifa bo'yicha natijalar" })
    const nginxRow = table.getByRole('row').filter({ has: page.getByRole('rowheader', { name: NGINX.title }) })
    await expect(nginxRow).toContainText('Yakunlangan')
    await expect(nginxRow.getByLabel(`${NGINX.checks} ta tekshiruvdan ${NGINX.checks} tasi o'tdi`)).toHaveText(
      `${NGINX.checks}/${NGINX.checks}`
    )
    const dockerRow = table.getByRole('row').filter({ has: page.getByRole('rowheader', { name: DOCKER.title }) })
    await expect(dockerRow).toContainText('Yangi')
    await expect(dockerRow.getByRole('link', { name: DOCKER.title })).toBeVisible()

    // Skills: the "web" category (Uzbek label "Web") is above 0%.
    const web = page.getByRole('progressbar', { name: 'Web', exact: true })
    await expect(web).toBeVisible()
    expect(Number(await web.getAttribute('aria-valuenow'))).toBeGreaterThan(0)

    // Achievements: "first-blood" ("Birinchi qadam") is earned.
    const achievements = page.getByRole('region', { name: /Yutuqlar/ })
    const first = achievements.getByRole('listitem').filter({ hasText: 'Birinchi qadam' })
    await expect(first).toContainText('Olingan')
    await achievements.getByRole('button', { name: 'Olingan' }).click()
    await expect(achievements.getByRole('listitem').filter({ hasText: 'Birinchi qadam' })).toBeVisible()

    // Reset: the confirm button stays disabled until the username is typed.
    await page.getByRole('region', { name: 'Progressni tiklash' }).getByRole('button', { name: 'Progressni tiklash' }).click()
    const dialog = page.getByRole('dialog', { name: 'Progressni tiklash' })
    await expect(dialog).toBeVisible()
    const confirm = dialogButton(page, 'Progressni tiklash')
    await expect(confirm).toBeDisabled()
    const input = dialog.getByRole('textbox')
    await input.fill(`${student.username}-x`)
    await expect(confirm).toBeDisabled()
    await input.fill(student.username)
    await expect(confirm).toBeEnabled()
    const reset = page.waitForResponse((r) => r.url().endsWith('/api/progress') && r.request().method() === 'DELETE')
    await confirm.click()
    expect((await reset).status()).toBe(204)
    await expect(dialog).toHaveCount(0)

    await expect(overview).toContainText(/Yakunlangan\s*0\/6/)
    await expect(page.getByText('Hali natija yo\'q')).toBeVisible()
    await expect(page.getByRole('region', { name: 'Progressni tiklash' }).getByRole('button', { name: 'Progressni tiklash' })).toBeDisabled()

    const progress = await (await studentApi.get('/api/progress')).json()
    expect(progress.completed).toBe(0)
    expect(progress.totalScore).toBe(0)

    // Task 2 is locked again.
    await page.goto(`/tasks?task=${DOCKER.id}`)
    const dockerCard = page
      .getByRole('region', { name: "Vazifalar ro'yxati" })
      .getByRole('listitem')
      .getByRole('button')
      .filter({ hasText: DOCKER.title })
    await expect(dockerCard).toContainText('Qulflangan')
    await expect(
      page.getByRole('complementary', { name: DOCKER.title }).getByText('Ochish uchun avvalgi vazifalarni yakunlang')
    ).toBeVisible()
    const tasks = (await (await studentApi.get('/api/tasks')).json()) as { id: string; unlocked: boolean; completed: boolean }[]
    expect(tasks.find((t) => t.id === DOCKER.id)?.unlocked).toBe(false)
    expect(tasks.find((t) => t.id === NGINX.id)?.completed).toBe(false)
  })

  test('a new student sees an empty progress page with reset disabled', async ({ page, student }) => {
    void student
    await page.goto('/progress')
    await expect(page.getByRole('region', { name: "Umumiy ko'rsatkichlar" })).toContainText(/Yakunlangan\s*0\/6/)
    await expect(page.getByText("Hali natija yo'q")).toBeVisible()
    await expect(page.getByRole('progressbar', { name: 'Web', exact: true })).toHaveAttribute('aria-valuenow', '0')
    await expect(
      page.getByRole('region', { name: /Yutuqlar/ }).getByRole('listitem').filter({ hasText: 'Birinchi qadam' })
    ).not.toContainText('Olingan')
    await expect(page.getByRole('region', { name: 'Progressni tiklash' }).getByRole('button', { name: 'Progressni tiklash' })).toBeDisabled()
  })
})
