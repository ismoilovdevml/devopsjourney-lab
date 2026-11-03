export const ADMIN_STATE = '.auth/admin.json'

export function adminCredentials(): { username: string; password: string } {
  const username = process.env.ADMIN_USERNAME ?? 'admin'
  const password = process.env.ADMIN_PASSWORD
  if (!password) {
    throw new Error('ADMIN_PASSWORD is not set. Run: set -a; . ../.env; set +a')
  }
  return { username, password }
}

/** State-changing API calls must carry an allowed Origin (OriginGuard). */
export function apiHeaders(baseURL: string): Record<string, string> {
  return { Origin: new URL(baseURL).origin }
}
