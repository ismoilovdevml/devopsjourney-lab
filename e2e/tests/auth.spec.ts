import { test, expect, newStudent } from '../fixtures'
import {
  authForm,
  createStudentAccount,
  expectLoginRedirect,
  headerProfileLink,
  loginViaUi,
  pageTitle,
  submitLogin
} from '../fixtures/auth'

const INVALID_CREDENTIALS = "Login yoki parol noto'g'ri"

test.describe('register', () => {
  test('new student registers through the UI and lands on the dashboard', async ({ page }) => {
    const student = newStudent('e2e-reg')
    await page.goto('/login')
    await expect(page).toHaveTitle(pageTitle('Kirish'))

    const form = authForm(page)
    await form.registerTab.click()
    await expect(form.registerTab).toHaveAttribute('aria-selected', 'true')
    await expect(form.loginTab).toHaveAttribute('aria-selected', 'false')
    await expect(page.getByRole('heading', { name: 'Hisob yarating' })).toBeVisible()

    await form.username.fill(student.username)
    await form.email.fill(student.email)
    await form.password.fill(student.password)

    const registered = page.waitForResponse((r) => r.url().endsWith('/api/auth/register'))
    await form.registerSubmit.click()
    expect((await registered).status()).toBe(201)

    await expect(page).toHaveURL(/\/$/)
    await expect(page).toHaveTitle(pageTitle('Bosh sahifa'))
    await expect(headerProfileLink(page, student.username)).toBeVisible()
    await expect(page.getByText(`${student.username}@lab`)).toBeVisible()
  })

  test('the "Ro\'yxatdan o\'ting" link and ?mode=register open the register tab', async ({ page }) => {
    await page.goto('/login')
    const form = authForm(page)
    await page.getByRole('button', { name: "Ro'yxatdan o'ting" }).click()
    await expect(form.registerTab).toHaveAttribute('aria-selected', 'true')
    await expect(form.email).toBeVisible()

    await page.getByRole('button', { name: 'Kiring', exact: true }).click()
    await expect(form.loginTab).toHaveAttribute('aria-selected', 'true')
    await expect(form.email).toHaveCount(0)

    await page.goto('/login?mode=register')
    await expect(form.registerTab).toHaveAttribute('aria-selected', 'true')
  })

  test('client-side validation flags each invalid field without calling the API', async ({ page }) => {
    const authCalls: string[] = []
    page.on('request', (r) => {
      if (r.url().includes('/api/auth/')) authCalls.push(r.url())
    })

    await page.goto('/login?mode=register')
    const form = authForm(page)
    await expect(form.registerTab).toHaveAttribute('aria-selected', 'true')

    // Fields start valid: nothing was touched yet.
    await expect(form.username).toHaveAttribute('aria-invalid', 'false')

    // Bad username (space + punctuation), invalid email, short password.
    await form.username.fill('bad user!')
    await form.email.fill('not-an-email')
    await form.password.fill('short1')
    await form.registerSubmit.click()

    await expect(form.username).toHaveAttribute('aria-invalid', 'true')
    await expect(form.username).toHaveAccessibleDescription(/Faqat kichik lotin harflari/)
    await expect(form.username).toBeFocused()

    await expect(form.email).toHaveAttribute('aria-invalid', 'true')
    await expect(form.email).toHaveAccessibleDescription("Email manzili noto'g'ri")

    await expect(form.password).toHaveAttribute('aria-invalid', 'true')
    await expect(form.password).toHaveAccessibleDescription("Kamida 8 ta belgi bo'lishi kerak")

    // Too short a username has its own message.
    await form.username.fill('ab')
    await expect(form.username).toHaveAccessibleDescription("3–32 ta belgi bo'lishi kerak")

    // Password equal to the username.
    await form.username.fill('valid_user1')
    await form.email.fill('valid_user1@example.test')
    await form.password.fill('valid_user1')
    await expect(form.username).toHaveAttribute('aria-invalid', 'false')
    await expect(form.email).toHaveAttribute('aria-invalid', 'false')
    await expect(form.password).toHaveAttribute('aria-invalid', 'true')
    await expect(form.password).toHaveAccessibleDescription('Parol foydalanuvchi nomi bilan bir xil bo‘lmasin')
    await form.registerSubmit.click()
    await expect(form.password).toBeFocused()

    await expect(page).toHaveURL(/\/login/)
    expect(authCalls, 'invalid forms must not reach the rate-limited auth API').toEqual([])
  })

  test('validation errors appear on blur and clear once the value is fixed', async ({ page }) => {
    await page.goto('/login?mode=register')
    const form = authForm(page)

    await form.email.fill('nope@')
    await form.email.blur()
    await expect(form.email).toHaveAttribute('aria-invalid', 'true')

    await form.email.fill('fixed@example.test')
    await expect(form.email).toHaveAttribute('aria-invalid', 'false')
    await expect(form.email).not.toHaveAccessibleDescription("Email manzili noto'g'ri")
  })

  test('duplicate username is rejected with a 409 "band" message', async ({ page, baseURL }) => {
    const existing = await createStudentAccount(baseURL as string, 'e2e-dup')

    await page.goto('/login?mode=register')
    const form = authForm(page)
    await form.username.fill(existing.username)
    await form.email.fill(`other-${existing.email}`)
    await form.password.fill(existing.password)

    const response = page.waitForResponse((r) => r.url().endsWith('/api/auth/register'))
    await form.registerSubmit.click()
    expect((await response).status()).toBe(409)

    await expect(form.formAlert).toBeVisible()
    await expect(form.formAlert).toContainText('band')
    await expect(page).toHaveURL(/\/login/)

    // Editing any field clears the stale server error.
    await form.username.fill(`${existing.username}x`)
    await expect(form.formAlert).toHaveCount(0)
  })
})

test.describe('login', () => {
  test('student logs in through the UI with correct credentials', async ({ page, baseURL }) => {
    const student = await createStudentAccount(baseURL as string, 'e2e-login')

    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Qaytganingizdan xursandmiz' })).toBeVisible()
    const response = page.waitForResponse((r) => r.url().endsWith('/api/auth/login'))
    await submitLogin(page, student.username, student.password)
    expect((await response).status()).toBe(200)

    await expect(page).toHaveURL(/\/$/)
    await expect(headerProfileLink(page, student.username)).toBeVisible()
  })

  test('wrong password and unknown user get the same generic error', async ({ page, baseURL }) => {
    const student = await createStudentAccount(baseURL as string, 'e2e-badpw')
    const form = authForm(page)

    await page.goto('/login')
    let response = page.waitForResponse((r) => r.url().endsWith('/api/auth/login'))
    await submitLogin(page, student.username, `${student.password}-wrong`)
    expect((await response).status()).toBe(401)
    await expect(form.formAlert).toHaveText(INVALID_CREDENTIALS)
    const wrongPasswordBody = await (await response).text()

    response = page.waitForResponse((r) => r.url().endsWith('/api/auth/login'))
    await form.username.fill(`ghost-${student.username}`)
    await form.password.fill(student.password)
    await form.loginSubmit.click()
    expect((await response).status()).toBe(401)
    await expect(form.formAlert).toHaveText(INVALID_CREDENTIALS)
    const unknownUserBody = await (await response).text()

    // The API must not reveal whether the username exists either.
    expect(unknownUserBody).toBe(wrongPasswordBody)
    await expect(page).toHaveURL(/\/login/)
  })

  test('password visibility toggle switches the input type', async ({ page }) => {
    await page.goto('/login')
    const form = authForm(page)
    await form.password.fill('secret-value-123')
    await expect(form.password).toHaveAttribute('type', 'password')

    const show = page.getByRole('button', { name: 'Parolni ko‘rsatish' })
    await expect(show).toHaveAttribute('aria-pressed', 'false')
    await show.click()
    await expect(form.password).toHaveAttribute('type', 'text')
    await expect(form.password).toHaveValue('secret-value-123')

    const hide = page.getByRole('button', { name: 'Parolni yashirish' })
    await expect(hide).toHaveAttribute('aria-pressed', 'true')
    await hide.click()
    await expect(form.password).toHaveAttribute('type', 'password')
  })

  test('logged-in user visiting /login is sent to the dashboard (or the redirect target)', async ({ page, student }) => {
    await page.goto('/login')
    await expect(page).toHaveURL(/\/$/)
    await expect(headerProfileLink(page, student.username)).toBeVisible()

    await page.goto('/login?redirect=%2Fleaderboard')
    await expect(page).toHaveURL(/\/leaderboard$/)
  })
})

test.describe('session', () => {
  test('auth cookie is httpOnly and the session survives a reload', async ({ page, context, student }) => {
    await page.goto('/')
    await expect(headerProfileLink(page, student.username)).toBeVisible()

    const cookie = (await context.cookies()).find((c) => c.name === 'dj_token')
    expect(cookie, 'dj_token cookie must be set').toBeDefined()
    expect(cookie?.httpOnly).toBe(true)
    expect(cookie?.sameSite).toBe('Strict')

    expect(await page.evaluate(() => document.cookie)).not.toContain('dj_token')

    await page.reload()
    await expect(page).toHaveURL(/\/$/)
    await expect(headerProfileLink(page, student.username)).toBeVisible()
  })

  test('logout from the header returns to /login and protects routes again', async ({ page, context, student }) => {
    await page.goto('/')
    await expect(headerProfileLink(page, student.username)).toBeVisible()

    const logout = page.waitForResponse((r) => r.url().endsWith('/api/auth/logout'))
    await page.getByRole('banner').getByRole('button', { name: 'Chiqish' }).click()
    expect((await logout).status()).toBe(204)
    await expect(page).toHaveURL(/\/login$/)

    expect((await context.cookies()).find((c) => c.name === 'dj_token')).toBeUndefined()
    expect((await page.request.get('/api/me')).status()).toBe(401)

    await page.goto('/progress')
    await expectLoginRedirect(page, '/progress')
    await page.goto('/')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('logout from the profile page returns to /login', async ({ page, student }) => {
    await page.goto('/profile')
    await expect(headerProfileLink(page, student.username)).toBeVisible()

    await page
      .getByRole('navigation', { name: "Sozlamalar bo'limlari" })
      .getByRole('button', { name: 'Chiqish' })
      .click()
    await expect(page).toHaveURL(/\/login$/)

    await page.goto('/profile?tab=security')
    await expectLoginRedirect(page, '/profile?tab=security')
  })

  test('deep link while logged out returns to the page after login', async ({ page, baseURL }) => {
    const student = await createStudentAccount(baseURL as string, 'e2e-deep')

    await page.goto('/progress')
    await expectLoginRedirect(page, '/progress')
    await submitLogin(page, student.username, student.password)
    await expect(page).toHaveURL(/\/progress$/)
    await expect(page).toHaveTitle(pageTitle('Natijalar'))
  })

  test('redirect keeps the query string of the original page', async ({ page, baseURL }) => {
    const student = await createStudentAccount(baseURL as string, 'e2e-deepq')

    await page.goto('/leaderboard?period=week')
    await expectLoginRedirect(page, '/leaderboard?period=week')
    await submitLogin(page, student.username, student.password)
    await expect(page).toHaveURL(/\/leaderboard\?period=week$/)
    await expect(page.getByRole('group', { name: 'Davr' }).getByRole('button', { name: 'Bu hafta' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
  })

  test('off-site redirect targets are ignored after login', async ({ page, baseURL }) => {
    const student = await createStudentAccount(baseURL as string, 'e2e-openredir')

    for (const target of ['//evil.example/', 'https://evil.example/', '/\\evil.example']) {
      await page.goto(`/login?redirect=${encodeURIComponent(target)}`)
      await submitLogin(page, student.username, student.password)
      await expect(page).toHaveURL(new URL('/', baseURL).href)
      await page.getByRole('banner').getByRole('button', { name: 'Chiqish' }).click()
      await expect(page).toHaveURL(/\/login$/)
    }
  })

  test('login via UI after an explicit logout works again', async ({ page, student }) => {
    await page.goto('/')
    await page.getByRole('banner').getByRole('button', { name: 'Chiqish' }).click()
    await expect(page).toHaveURL(/\/login$/)

    await loginViaUi(page, student)
    await expect(page).toHaveURL(/\/$/)
    await expect(headerProfileLink(page, student.username)).toBeVisible()
  })
})
