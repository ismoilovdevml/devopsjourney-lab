import { test as base, expect, type APIRequestContext, type Page } from '@playwright/test'
import { randomBytes } from 'node:crypto'
import { ADMIN_STATE, apiHeaders } from './env'

export interface Student {
  username: string
  email: string
  password: string
}

export function newStudent(prefix = 'e2e'): Student {
  const id = randomBytes(4).toString('hex')
  const username = `${prefix}-${id}`
  return { username, email: `${username}@example.test`, password: `Pw-${randomBytes(8).toString('hex')}` }
}

/** Registers a student through the API; the response sets the auth cookie on `request`. */
export async function registerStudent(request: APIRequestContext, student: Student): Promise<void> {
  const res = await request.post('/api/auth/register', {
    data: { username: student.username, email: student.email, password: student.password }
  })
  expect(res.status(), await res.text()).toBe(201)
}

/** Stops the student's active lab session, if any, so the global session cap is not exhausted. */
export async function stopActiveSession(request: APIRequestContext): Promise<void> {
  const res = await request.get('/api/sessions/active')
  if (res.status() !== 200) return
  const { sessionId } = (await res.json()) as { sessionId: string }
  await request.delete(`/api/sessions/${sessionId}`)
}

type Fixtures = {
  /** A fresh student, registered and logged in on `page` (and on `studentApi`). */
  student: Student
  studentApi: APIRequestContext
  /** A page logged in as the seeded admin. */
  adminPage: Page
}

export const test = base.extend<Fixtures>({
  extraHTTPHeaders: async ({ baseURL }, use) => {
    await use(apiHeaders(baseURL as string))
  },
  student: async ({ page }, use) => {
    const student = newStudent()
    await registerStudent(page.request, student)
    await use(student)
    await stopActiveSession(page.request)
  },
  studentApi: async ({ page, student }, use) => {
    void student
    await use(page.request)
  },
  adminPage: async ({ browser, baseURL }, use) => {
    const context = await browser.newContext({
      storageState: ADMIN_STATE,
      baseURL,
      extraHTTPHeaders: apiHeaders(baseURL as string)
    })
    const page = await context.newPage()
    await use(page)
    await context.close()
  }
})

export { expect }
