import { request as playwrightRequest } from '@playwright/test'
import { randomBytes } from 'node:crypto'
import { test, expect } from '../fixtures'
import { apiHeaders } from '../fixtures/env'
import { toast } from '../fixtures/admin'

test.describe('profile', () => {
  test('email, GitHub and goal: save is disabled until dirty, then persists', async ({ page, student, studentApi }) => {
    await page.goto('/profile')
    await expect(page.getByRole('heading', { level: 1, name: student.username })).toBeVisible()

    const email = page.getByLabel('Email')
    const github = page.getByLabel('GitHub')
    const goal = page.getByLabel('Maqsad')
    const save = page.getByRole('button', { name: 'Saqlash' })
    const cancel = page.getByRole('button', { name: 'Bekor qilish' })

    await expect(page.getByLabel('Foydalanuvchi nomi')).toHaveValue(student.username)
    await expect(email).toHaveValue(student.email)
    await expect(save).toBeDisabled()
    await expect(cancel).toBeDisabled()

    const next = {
      email: `new-${student.email}`,
      github: `octo-${randomBytes(3).toString('hex')}`,
      goal: "6 oyda Junior DevOps Engineer bo'lish"
    }
    await email.fill(next.email)
    await expect(save).toBeEnabled()
    await expect(page.getByText("Saqlanmagan o'zgarishlar bor")).toBeVisible()
    // Reverting the only change makes the form clean again.
    await email.fill(student.email)
    await expect(save).toBeDisabled()

    await email.fill(next.email)
    await github.fill(`https://github.com/${next.github}`)
    await goal.fill(next.goal)
    await expect(page.getByText(`${next.goal.length}/200`)).toBeVisible()

    const put = page.waitForResponse((r) => r.request().method() === 'PUT' && new URL(r.url()).pathname === '/api/me')
    await save.click()
    const res = await put
    expect(res.status()).toBe(200)
    // The client normalizes a profile URL to the bare username before sending it.
    expect(res.request().postDataJSON()).toEqual(next)
    await expect(toast(page, 'Profil saqlandi')).toBeVisible()
    await expect(save).toBeDisabled()
    await expect(github).toHaveValue(next.github)

    await page.reload()
    await expect(email).toHaveValue(next.email)
    await expect(github).toHaveValue(next.github)
    await expect(goal).toHaveValue(next.goal)
    await expect(save).toBeDisabled()

    const me = await (await studentApi.get('/api/me')).json()
    expect(me).toMatchObject(next)
  })

  test('an invalid GitHub username shows an inline error and is not sent', async ({ page, student, studentApi }) => {
    void student
    await page.goto('/profile')
    const github = page.getByLabel('GitHub')
    await github.fill('bad_name!')
    await github.blur()

    const error = page.locator('#pf-github-err')
    await expect(error).toHaveText("GitHub foydalanuvchi nomi: harf, raqam va '-', ko'pi bilan 39 belgi.")
    await expect(github).toHaveAttribute('aria-invalid', 'true')
    await expect(github).toHaveAttribute('aria-describedby', 'pf-github-err')

    let sent = false
    page.on('request', (r) => {
      if (r.method() === 'PUT' && new URL(r.url()).pathname === '/api/me') sent = true
    })
    await page.getByRole('button', { name: 'Saqlash' }).click()
    await expect(github).toBeFocused()
    await page.waitForTimeout(300)
    expect(sent).toBe(false)
    expect(((await (await studentApi.get('/api/me')).json()) as { github: string }).github).toBe('')

    // Fixing the value clears the error.
    await github.fill('valid-name')
    await expect(error).toHaveCount(0)
  })

  test('terminal theme picker: arrow keys change the preference and it survives a reload', async ({ page, student }) => {
    void student
    await page.goto('/profile?tab=appearance')
    const group = page.getByRole('radiogroup', { name: 'Terminal mavzusi' })
    const radio = (name: string) => group.getByRole('radio', { name, exact: true })
    const stored = () => page.evaluate(() => JSON.parse(localStorage.getItem('dj.prefs') ?? 'null'))

    await expect(radio('Matrix')).toHaveAttribute('aria-checked', 'true')
    await expect(group.getByRole('radio')).toHaveCount(4)

    await radio('Matrix').focus()
    await page.keyboard.press('ArrowRight')
    await expect(radio('Ocean')).toHaveAttribute('aria-checked', 'true')
    await expect(radio('Ocean')).toBeFocused()
    await expect(radio('Matrix')).toHaveAttribute('aria-checked', 'false')
    await expect(radio('Ocean')).toHaveAttribute('tabindex', '0')
    await expect(radio('Matrix')).toHaveAttribute('tabindex', '-1')
    await expect.poll(stored).toMatchObject({ terminalTheme: 'ocean' })

    await page.keyboard.press('ArrowDown')
    await expect(radio('Dracula')).toHaveAttribute('aria-checked', 'true')
    await page.keyboard.press('ArrowLeft')
    await expect(radio('Ocean')).toHaveAttribute('aria-checked', 'true')
    await page.keyboard.press('End')
    await expect(radio('Amber CRT')).toHaveAttribute('aria-checked', 'true')
    await expect(radio('Amber CRT')).toBeFocused()
    await expect.poll(stored).toMatchObject({ terminalTheme: 'amber' })

    // The live preview uses the selected theme's background (#140F06).
    await expect(page.getByRole('img', { name: "Terminal ko'rinishi namunasi" })).toHaveCSS('background-color', 'rgb(20, 15, 6)')

    await page.reload()
    await expect(radio('Amber CRT')).toHaveAttribute('aria-checked', 'true')
    await expect(radio('Matrix')).toHaveAttribute('aria-checked', 'false')
    expect(await stored()).toMatchObject({ terminalTheme: 'amber' })
  })

  test('password change: wrong current, weak and success; old sessions are revoked', async ({
    page,
    student,
    browser,
    baseURL
  }) => {
    // A second browser that logged in with the old password before the change.
    const other = await browser.newContext({ baseURL, extraHTTPHeaders: apiHeaders(baseURL!) })
    try {
      const otherLogin = await other.request.post('/api/auth/login', {
        data: { username: student.username, password: student.password }
      })
      expect(otherLogin.status()).toBe(200)
      expect((await other.request.get('/api/me')).status()).toBe(200)
      // JWT "iat" has one-second precision; make sure the change happens in a later second.
      await page.waitForTimeout(1_100)

      await page.goto('/profile?tab=security')
      const current = page.getByLabel('Joriy parol')
      const next = page.getByLabel('Yangi parol', { exact: true })
      const confirm = page.getByLabel('Yangi parolni tasdiqlang')
      const submit = page.getByRole('button', { name: 'Parolni yangilash' })
      const newPassword = `New-${randomBytes(8).toString('hex')}!`

      // Wrong current password → server 401 shown on the field; the page stays logged in.
      await current.fill('definitely-not-it')
      await next.fill(newPassword)
      await confirm.fill(newPassword)
      const wrong = page.waitForResponse((r) => new URL(r.url()).pathname === '/api/me/password')
      await submit.click()
      expect((await wrong).status()).toBe(401)
      await expect(page.locator('#pw-current-err')).toHaveText("Joriy parol noto'g'ri.")
      await expect(current).toBeFocused()
      await expect(page).toHaveURL(/\/profile/)

      // Weak passwords are rejected with an inline error.
      await current.fill(student.password)
      await next.fill('short')
      await confirm.fill('short')
      await submit.click()
      await expect(page.locator('#pw-new-err')).toHaveText("Parol kamida 8 belgidan iborat bo'lishi kerak.")
      await expect(page.locator('#pw-strength')).toContainText('Zaif')
      await next.fill('password123')
      await confirm.fill('password123')
      await submit.click()
      await expect(page.locator('#pw-new-err')).toHaveText("Bu parol juda keng tarqalgan — boshqasini tanlang.")

      // Mismatched confirmation.
      await next.fill(newPassword)
      await confirm.fill(`${newPassword}x`)
      await submit.click()
      await expect(page.locator('#pw-confirm-err')).toHaveText('Parollar mos kelmadi.')

      // Success.
      await confirm.fill(newPassword)
      const ok = page.waitForResponse((r) => new URL(r.url()).pathname === '/api/me/password')
      await submit.click()
      expect((await ok).status()).toBe(204)
      await expect(toast(page, 'Parol yangilandi')).toBeVisible()
      await expect(current).toHaveValue('')
      await expect(next).toHaveValue('')

      // The other browser's token predates the change → revoked.
      expect((await other.request.get('/api/me')).status()).toBe(401)
      const otherPage = await other.newPage()
      await otherPage.goto('/progress')
      await expect(otherPage).toHaveURL(/\/login/)

      // This page keeps working (its cookie was re-issued).
      await page.reload()
      await expect(page).toHaveURL(/\/profile/)
      await expect(page.getByRole('heading', { level: 1, name: student.username })).toBeVisible()
      expect((await page.request.get('/api/me')).status()).toBe(200)

      // Old password no longer works; the new one does.
      const fresh = await playwrightRequest.newContext({ baseURL, extraHTTPHeaders: apiHeaders(baseURL!) })
      try {
        const old = await fresh.post('/api/auth/login', { data: { username: student.username, password: student.password } })
        expect(old.status()).toBe(401)
        const good = await fresh.post('/api/auth/login', { data: { username: student.username, password: newPassword } })
        expect(good.status()).toBe(200)
      } finally {
        await fresh.dispose()
      }

      // And through the UI login form.
      await otherPage.getByLabel('Foydalanuvchi nomi').fill(student.username)
      await otherPage.getByLabel('Parol', { exact: true }).fill(newPassword)
      await otherPage.getByRole('button', { name: 'Kirish', exact: true }).click()
      await expect(otherPage).toHaveURL(/\/progress$/)
    } finally {
      await other.close()
    }
  })
})
