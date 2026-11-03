import http from 'node:http'
import { randomBytes } from 'node:crypto'
import type { APIRequestContext } from '@playwright/test'
import { test, expect } from '../fixtures'
import { firstTaskId, studentClient } from '../fixtures/auth'

const EVIL_ORIGIN = 'https://evil.example'

/** Collects every object key in a JSON value, recursively. */
function allKeys(value: unknown, out = new Set<string>()): Set<string> {
  if (Array.isArray(value)) value.forEach((v) => allKeys(v, out))
  else if (value && typeof value === 'object') {
    for (const [k, v] of Object.entries(value)) {
      out.add(k)
      allKeys(v, out)
    }
  }
  return out
}

/**
 * Sends a raw WebSocket upgrade request and resolves with the HTTP status of the answer
 * (101 when the server accepted the upgrade). Playwright's request API cannot upgrade, and
 * browsers do not allow overriding Origin, so this goes through node:http.
 */
function wsHandshakeStatus(baseURL: string, path: string, headers: Record<string, string>): Promise<number> {
  const url = new URL(path, baseURL)
  return new Promise((resolve, reject) => {
    const req = http.request({
      hostname: url.hostname,
      port: url.port,
      path: `${url.pathname}${url.search}`,
      headers: {
        Connection: 'Upgrade',
        Upgrade: 'websocket',
        'Sec-WebSocket-Version': '13',
        'Sec-WebSocket-Key': randomBytes(16).toString('base64'),
        ...headers
      },
      timeout: 15_000
    })
    req.on('upgrade', (res, socket) => {
      socket.destroy()
      resolve(res.statusCode ?? 0)
    })
    req.on('response', (res) => {
      res.resume()
      resolve(res.statusCode ?? 0)
    })
    req.on('timeout', () => req.destroy(new Error('WebSocket handshake timed out')))
    req.on('error', reject)
    req.end()
  })
}

async function cookieHeader(api: APIRequestContext): Promise<string> {
  const { cookies } = await api.storageState()
  const token = cookies.find((c) => c.name === 'dj_token')
  expect(token, 'dj_token cookie').toBeDefined()
  return `dj_token=${token?.value}`
}

test.describe('security headers', () => {
  for (const path of ['/', '/login', '/tasks', '/no/such/page']) {
    test(`HTML document ${path} carries CSP, X-Frame-Options and nosniff`, async ({ page }) => {
      const res = await page.goto(path)
      expect(res, 'navigation response').not.toBeNull()
      const headers = await res!.allHeaders()
      expect(headers['content-type']).toContain('text/html')

      const csp = headers['content-security-policy'] ?? ''
      expect(csp).toContain("default-src 'self'")
      expect(csp).toContain("script-src 'self'")
      expect(csp).not.toMatch(/script-src[^;]*'unsafe-(inline|eval)'/)
      expect(csp).toContain("frame-ancestors 'none'")
      expect(csp).toContain("object-src 'none'")
      expect(csp).toContain("base-uri 'self'")

      expect(headers['x-frame-options']).toBe('DENY')
      expect(headers['x-content-type-options']).toBe('nosniff')
      expect(headers['referrer-policy']).toBe('strict-origin-when-cross-origin')
    })
  }

  test('the app runs without CSP violations', async ({ page, student }) => {
    void student
    await page.addInitScript(() => {
      const w = window as unknown as { __cspViolations: string[] }
      w.__cspViolations = []
      document.addEventListener('securitypolicyviolation', (e) => {
        w.__cspViolations.push(`${e.violatedDirective} ${e.blockedURI}`)
      })
    })
    for (const path of ['/', '/tasks', '/progress', '/leaderboard', '/profile']) {
      await page.goto(path)
      await page.waitForLoadState('networkidle')
      const violations = await page.evaluate(() => (window as unknown as { __cspViolations: string[] }).__cspViolations)
      expect(violations, `CSP violations on ${path}`).toEqual([])
    }
  })

  test('API responses are JSON with nosniff and no-store', async ({ request }) => {
    const res = await request.get('/api/tasks')
    expect(res.status()).toBe(200)
    const headers = res.headers()
    expect(headers['content-type']).toContain('application/json')
    expect(headers['x-content-type-options']).toContain('nosniff')
    expect(headers['cache-control']).toContain('no-store')
  })
})

test.describe('task data never leaks solutions', () => {
  for (const who of ['anonymous', 'student'] as const) {
    test(`/api/tasks and /api/tasks/:id have no cmd/setup keys (${who})`, async ({ request, baseURL }) => {
      let api = request
      let dispose: (() => Promise<void>) | undefined
      if (who === 'student') {
        const client = await studentClient(baseURL as string, 'e2e-leak')
        api = client.api
        dispose = () => client.api.dispose()
      }
      try {
        const list = await api.get('/api/tasks')
        expect(list.status()).toBe(200)
        const tasks = (await list.json()) as { id: string }[]
        expect(tasks.length).toBeGreaterThan(0)
        const listKeys = allKeys(tasks)
        expect(listKeys.has('cmd')).toBe(false)
        expect(listKeys.has('setup')).toBe(false)

        for (const { id } of tasks) {
          const detail = await api.get(`/api/tasks/${id}`)
          expect(detail.status(), id).toBe(200)
          const body = await detail.json()
          const keys = allKeys(body)
          expect(keys.has('cmd'), `${id} exposes checks[].cmd`).toBe(false)
          expect(keys.has('setup'), `${id} exposes setup`).toBe(false)
          // checks are reduced to names only.
          for (const check of (body as { checks?: object[] }).checks ?? []) {
            expect(Object.keys(check)).toEqual(['name'])
          }
        }
      } finally {
        await dispose?.()
      }
    })
  }
})

test.describe('origin guard (CSRF)', () => {
  test('state-changing requests from a foreign Origin are rejected with 403', async ({ baseURL }) => {
    const { api } = await studentClient(baseURL as string, 'e2e-csrf')
    try {
      const evil = { headers: { Origin: EVIL_ORIGIN } }

      const logout = await api.post('/api/auth/logout', evil)
      expect(logout.status()).toBe(403)

      const taskId = await firstTaskId(api)
      const createSession = await api.post('/api/sessions', { ...evil, data: { taskId } })
      expect(createSession.status()).toBe(403)

      expect((await api.put('/api/me', { ...evil, data: { goal: 'pwned' } })).status()).toBe(403)
      expect((await api.delete('/api/progress', evil)).status()).toBe(403)
      expect(
        (await api.post('/api/me/password', { ...evil, data: { current_password: 'x', new_password: 'y' } })).status()
      ).toBe(403)

      // A foreign Referer without Origin is treated the same way.
      const refererOnly = await api.post('/api/auth/logout', {
        headers: { Origin: '', Referer: `${EVIL_ORIGIN}/attack.html` }
      })
      expect(refererOnly.status()).toBe(403)

      // None of the rejected calls had an effect: still logged in, no session, goal untouched.
      const me = await api.get('/api/me')
      expect(me.status()).toBe(200)
      expect(((await me.json()) as { goal: string }).goal).not.toBe('pwned')
      expect((await api.get('/api/sessions/active')).status()).toBe(204)

      // Positive control: the same logout from the app's own origin works.
      expect((await api.post('/api/auth/logout')).status()).toBe(204)
      expect((await api.get('/api/me')).status()).toBe(401)
    } finally {
      await api.dispose()
    }
  })

  test('login from a foreign Origin is rejected (login CSRF)', async ({ baseURL }) => {
    const { student, api } = await studentClient(baseURL as string, 'e2e-logincsrf')
    try {
      const res = await api.post('/api/auth/login', {
        headers: { Origin: EVIL_ORIGIN },
        data: { username: student.username, password: student.password }
      })
      expect(res.status()).toBe(403)
      expect(res.headers()['set-cookie'] ?? '').not.toContain('dj_token')
    } finally {
      await api.dispose()
    }
  })

  test('CORS does not grant a foreign origin read access', async ({ request }) => {
    const res = await request.get('/api/tasks', { headers: { Origin: EVIL_ORIGIN } })
    expect(res.headers()['access-control-allow-origin']).toBeUndefined()
    expect(res.headers()['access-control-allow-credentials']).toBeUndefined()
  })
})

test.describe('lab session isolation', () => {
  let alice: APIRequestContext
  let bob: APIRequestContext
  let bobSessionId: string
  let base: string

  test.beforeAll(async ({}, testInfo) => {
    base = testInfo.project.use.baseURL as string
    alice = (await studentClient(base, 'e2e-alice')).api
    bob = (await studentClient(base, 'e2e-bob')).api

    const taskId = await firstTaskId(bob)
    const res = await bob.post('/api/sessions', { data: { taskId }, timeout: 60_000 })
    expect(res.status(), await res.text()).toBe(201)
    bobSessionId = ((await res.json()) as { sessionId: string }).sessionId
  })

  test.afterAll(async () => {
    if (bobSessionId) await bob?.delete(`/api/sessions/${bobSessionId}`)
    await alice?.dispose()
    await bob?.dispose()
  })

  test("student A cannot read, check, recreate or stop student B's session", async () => {
    const path = `/api/sessions/${bobSessionId}`

    expect((await alice.get(path)).status()).toBe(404)
    expect((await alice.post(`/api/check/${bobSessionId}`)).status()).toBe(404)
    expect((await alice.post(`${path}/recreate`)).status()).toBe(404)
    expect((await alice.delete(path)).status()).toBe(404)

    // Same answer as for an id that does not exist at all (no existence oracle).
    const missing = await alice.get('/api/sessions/00000000-0000-0000-0000-000000000000')
    expect(missing.status()).toBe(404)

    // Alice's own view is unaffected and Bob's session is still alive.
    expect((await alice.get('/api/sessions/active')).status()).toBe(204)
    const owned = await bob.get(path)
    expect(owned.status()).toBe(200)
    expect(((await owned.json()) as { sessionId: string }).sessionId).toBe(bobSessionId)
  })

  test('terminal WebSocket rejects a foreign Origin and other users', async () => {
    const wsPath = `/ws?sessionId=${encodeURIComponent(bobSessionId)}`
    const bobCookie = await cookieHeader(bob)
    const aliceCookie = await cookieHeader(alice)
    const ownOrigin = new URL(base).origin

    // Cross-site WebSocket hijacking: Bob's cookie, attacker's page.
    expect(await wsHandshakeStatus(base, wsPath, { Origin: EVIL_ORIGIN, Cookie: bobCookie })).toBe(403)
    // Another student's cookie from the right origin.
    expect(await wsHandshakeStatus(base, wsPath, { Origin: ownOrigin, Cookie: aliceCookie })).toBe(404)
    // No cookie at all.
    expect(await wsHandshakeStatus(base, wsPath, { Origin: ownOrigin })).toBe(401)
    // Positive control: the owner from the app's origin is upgraded.
    expect(await wsHandshakeStatus(base, wsPath, { Origin: ownOrigin, Cookie: bobCookie })).toBe(101)
  })
})

test.describe('not applicable', () => {
  test.skip('markdown XSS in task READMEs', () => {
    // Task READMEs are authored by admins in the repository (task.yaml + README.md) and rendered
    // with a sanitizer; students cannot submit markdown, so there is no student-controlled input to test.
  })
})
