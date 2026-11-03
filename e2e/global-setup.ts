import { request, type FullConfig } from '@playwright/test'
import { mkdirSync } from 'node:fs'
import { ADMIN_STATE, adminCredentials, apiHeaders } from './fixtures/env'

/** Logs the seeded admin in once and stores the cookie for admin specs. */
export default async function globalSetup(config: FullConfig): Promise<void> {
  const baseURL = config.projects[0]?.use.baseURL as string
  const { username, password } = adminCredentials()
  mkdirSync('.auth', { recursive: true })

  const ctx = await request.newContext({ baseURL, extraHTTPHeaders: apiHeaders(baseURL) })
  const health = await ctx.get('/readyz')
  if (!health.ok()) {
    throw new Error(`Stack is not ready at ${baseURL} (/readyz → ${health.status()}). Run 'make up' first.`)
  }
  const res = await ctx.post('/api/auth/login', { data: { username, password } })
  if (!res.ok()) {
    throw new Error(`Admin login failed (${res.status()}). Export ADMIN_USERNAME/ADMIN_PASSWORD from .env.`)
  }
  await ctx.storageState({ path: ADMIN_STATE })
  await ctx.dispose()
}
