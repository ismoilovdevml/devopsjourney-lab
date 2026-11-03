import { expect, request as playwrightRequest, type APIRequestContext, type Page } from '@playwright/test'
import { apiHeaders } from './env'
import { newStudent, registerStudent, type Student } from './index'

export const APP_NAME = 'DevOps Journey Lab'

/** Expected document.title for a route meta title. */
export function pageTitle(title: string): string {
  return `${title} · ${APP_NAME}`
}

/**
 * A standalone API client for a freshly registered student. It has its own cookie jar, so it never
 * logs the browser page in. Dispose it at the end of the test.
 */
export async function studentClient(
  baseURL: string,
  prefix = 'e2e'
): Promise<{ student: Student; api: APIRequestContext }> {
  const api = await playwrightRequest.newContext({ baseURL, extraHTTPHeaders: apiHeaders(baseURL) })
  const student = newStudent(prefix)
  await registerStudent(api, student)
  return { student, api }
}

/** Creates a student account through the API without touching any browser context. */
export async function createStudentAccount(baseURL: string, prefix = 'e2e'): Promise<Student> {
  const { student, api } = await studentClient(baseURL, prefix)
  await api.dispose()
  return student
}

/** Locators of the /login auth card. */
export function authForm(page: Page) {
  return {
    loginTab: page.getByRole('tab', { name: 'Kirish', exact: true }),
    registerTab: page.getByRole('tab', { name: "Ro'yxatdan o'tish", exact: true }),
    username: page.getByLabel('Foydalanuvchi nomi', { exact: true }),
    email: page.getByLabel('Email', { exact: true }),
    password: page.getByLabel('Parol', { exact: true }),
    loginSubmit: page.getByRole('button', { name: 'Kirish', exact: true }),
    registerSubmit: page.getByRole('button', { name: "Ro'yxatdan o'tish", exact: true }),
    formAlert: page.locator('form [role="alert"]')
  }
}

/** Fills and submits the login form on the current /login page. */
export async function submitLogin(page: Page, username: string, password: string): Promise<void> {
  const form = authForm(page)
  await expect(form.loginTab).toHaveAttribute('aria-selected', 'true')
  await form.username.fill(username)
  await form.password.fill(password)
  await form.loginSubmit.click()
}

/** Opens /login and logs in through the UI. */
export async function loginViaUi(page: Page, student: Pick<Student, 'username' | 'password'>): Promise<void> {
  await page.goto('/login')
  await submitLogin(page, student.username, student.password)
}

/** The current user's avatar link in the app header ("Profil: <username>"). */
export function headerProfileLink(page: Page, username: string) {
  return page.getByRole('banner').getByRole('link', { name: `Profil: ${username}` })
}

/** First task in learning-path order (always unlocked for a new student). */
export async function firstTaskId(api: APIRequestContext): Promise<string> {
  const res = await api.get('/api/tasks')
  expect(res.ok()).toBeTruthy()
  const tasks = (await res.json()) as { id: string; order: number }[]
  expect(tasks.length).toBeGreaterThan(0)
  return [...tasks].sort((a, b) => a.order - b.order)[0].id
}

/** Asserts the page was bounced to /login with `redirect` set to `target` (encoding-agnostic). */
export async function expectLoginRedirect(page: Page, target: string): Promise<void> {
  await expect
    .poll(() => {
      const url = new URL(page.url())
      return url.pathname === '/login' ? url.searchParams.get('redirect') : `not on /login: ${url.href}`
    })
    .toBe(target)
}
