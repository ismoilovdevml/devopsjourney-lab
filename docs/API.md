# API contract

All endpoints are under `/api` and return JSON. Errors use `{ "error": string, "message"?: string, "code": number }`.

## Authentication

Browser clients authenticate with an **httpOnly cookie** `dj_token` (`SameSite=Strict`, `Secure` unless
`COOKIE_SECURE=false`). The JWT is never returned to JavaScript. Non-browser clients may send
`Authorization: Bearer <jwt>` instead. Helpers live in `backend-go/internal/auth/cookie.go`
(`SetAuthCookie`, `ClearAuthCookie`, `TokenFromRequest`).

State-changing requests (`POST/PUT/PATCH/DELETE`) must come from an allowed origin: the backend rejects
them with `403` when an `Origin` header is present and not in `CORS_ORIGINS`.

| Method | Path | Auth | Body | Response |
|---|---|---|---|---|
| POST | `/api/auth/register` | — | `{username, email, password}` | `201 {user: User}` + sets cookie |
| POST | `/api/auth/login` | — | `{username, password}` | `200 {user: User}` + sets cookie |
| POST | `/api/auth/logout` | user | — | `204` + clears cookie |
| GET | `/api/me` | user | — | `200 User` |
| PUT | `/api/me` | user | `{email?, github?, goal?}` | `200 User` |
| POST | `/api/me/password` | user | `{current_password, new_password}` | `204` (`400` weak, `401` wrong current) |

Rules: username `^[a-z0-9_-]{3,32}$`; password 8–72 chars; login/register rate-limited per IP
(10/min); login failure messages never reveal whether the username exists.

```ts
User = { id, username, email, role: 'student'|'admin', active, github, goal, created_at }
```

## Tasks

| Method | Path | Auth | Response |
|---|---|---|---|
| GET | `/api/tasks` | optional | `TaskSummary[]` (sorted by `order`) |
| GET | `/api/tasks/:id` | optional | `TaskDetail` |

```ts
TaskSummary = { id, title, description, category, difficulty: 1|2|3, points, order,
                checks_count, unlocked, completed, in_progress }
TaskDetail  = TaskSummary & { image, readme /* markdown */, hints: string[], checks: {name}[] }
```

`checks[].cmd` and `setup` are **never** sent to clients (they reveal the solution).
`category` comes from `task.yaml` (`web`, `containers`, `services`, `users`, `network`, `automation`).
`points` comes from `task.yaml` (default `difficulty * 100`).

## Sessions (lab containers)

| Method | Path | Auth | Body | Response |
|---|---|---|---|---|
| POST | `/api/sessions` | user | `{taskId}` | `201 SessionResponse` (409 if task locked) |
| GET | `/api/sessions/active` | user | — | `200 SessionResponse` or `204` |
| GET | `/api/sessions/:id` | owner | — | `200 SessionResponse` |
| POST | `/api/sessions/:id/recreate` | owner | — | `201 SessionResponse` (old container removed) |
| DELETE | `/api/sessions/:id` | owner | — | `204` |
| POST | `/api/check/:sessionId` | owner | — | `200 CheckResponse` |
| WS | `/ws?sessionId=<id>` | owner (cookie) | — | terminal stream |

```ts
SessionResponse = { sessionId, taskId, ttlSeconds, expiresAt, createdAt }
CheckResponse   = { passed, score, total, results: {name, ok, output?}[] }
```

One active session per user: creating a new one stops the previous one. The WebSocket upgrade checks
the `Origin` header against `CORS_ORIGINS` and session ownership. Client→server frames:
raw text = stdin, `{"type":"resize","cols":n,"rows":n}` = resize.

## Progress & leaderboard

| Method | Path | Auth | Response |
|---|---|---|---|
| GET | `/api/progress` | user | `ProgressResponse` |
| DELETE | `/api/progress` | user | `204` (resets own progress) |
| GET | `/api/leaderboard?period=week\|month\|all&limit=50` | user | `LeaderboardResponse` |

```ts
ProgressResponse = {
  progress: { task_id, score, total, passed, attempts, time_spent, completed_at }[],
  totalScore, totalPossible, percentage, completed, totalTasks,
  timeInLabSeconds, streakDays,
  skills: { category, percent }[],
  achievements: { id, title, description, earned, earned_at? }[]
}
LeaderboardResponse = {
  entries: { user_id, username, score, completed, rank }[],
  me: { user_id, username, score, completed, rank } | null
}
```

## Admin (`role=admin`)

| Method | Path | Response |
|---|---|---|
| GET | `/api/admin/stats` | `{total_users, active_users, active_sessions, total_attempts, completion_rate, sessions_24h: number[24], task_completion: {task_id, attempts, passed}[]}` |
| GET | `/api/admin/users` | `AdminUser[]` (`User & {score, completed, last_active}`) |
| PUT | `/api/admin/users/:id` | `{role?, active?}` → `200 User` (admin cannot demote/deactivate self) |
| DELETE | `/api/admin/users/:id` | `204` (admin cannot delete self) |
| GET | `/api/admin/sessions` | `{id, user_id, username, task_id, status, created_at, expires_at}[]` |
| POST | `/api/admin/sessions/:id/terminate` | `204` |
| GET | `/api/admin/activity?limit=100` | `{id, username, action, details, ip_address, level, created_at}[]` |
| GET | `/api/admin/resources` | `{containers_running, containers_limit, sessions: {session_id, username, task_id, cpu_percent, mem_bytes, mem_limit_bytes}[]}` |
| POST | `/api/admin/tasks/reload` | `{count}` |

## Health

`GET /healthz` → `{status, timestamp, version, uptime}` (liveness).
`GET /readyz` → `200` when DB + Redis + Docker are reachable, else `503`.
