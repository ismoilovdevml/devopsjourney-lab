# DevOps Journey Lab — UI Design Spec (dark "server / terminal" theme)

This folder is a design handoff for the Vue 3 + TypeScript + Tailwind frontend in `frontend-vue/`.
The `*.dc.html` files are the visual reference for every page (open them as source: layout, copy,
colors and spacing are all inline). `assets/` holds the SVG images to ship with the app.

UI language: **Uzbek (Latin)**. Task titles stay in English as in `backend-go/tasks/*/task.yaml`.

---

## 1. Design tokens

### Colors
| Token | Hex | Use |
|---|---|---|
| `bg` | `#0A0E14` | Page background (under the wallpaper) |
| `surface` | `rgba(15,21,32,0.85)` / solid `#0F1520` | Cards, panels (slightly translucent over wallpaper) |
| `surface-2` | `#0B1018` | Inputs, inner tiles, terminal-like blocks |
| `surface-3` | `#141C2A` | Chips, avatar bg, active nav item |
| `raised` | `#1B2535` | Active segmented-control item |
| `terminal` | `#070A0F` | Terminal body |
| `border` | `#1F2A3A` | Card borders |
| `border-strong` | `#243247` / `#2A3850` | Inputs, buttons, dividers on hover |
| `divider` | `#1A2332` | Table row lines |
| `text` | `#E6EDF3` | Primary text |
| `text-2` | `#C9D4E0` | Secondary body text |
| `muted` | `#94A3B8` | Labels, meta (≥4.5:1 on surfaces) |
| `faint` | `#64748B` | Timestamps only (decorative) |
| `accent` | `#3DDC97` | Primary buttons, active state, XP, "done" |
| `accent-ink` | `#04130C` | Text on accent buttons |
| `info` | `#38BDF8` | "In progress", secondary data |
| `warn` | `#F5B342` | Warnings, rank, admin badge |
| `danger` | `#F87171` | Errors, destructive actions |
| `violet` | `#A78BFA` | Systemd / achievements |
| `rose` | `#FB7185` | Firewall |
| `teal` | `#2DD4BF` | Cron |

Per-task accent: nginx `#3DDC97`, docker `#38BDF8`, systemd `#A78BFA`, users `#F5B342`, firewall `#FB7185`, cron `#2DD4BF`.

Tinted badges: `color: X; background: X @ 10–12% alpha; border: 1px solid X @ 30–35% alpha`.
Glow (use sparingly — primary button, logo, live dots): `box-shadow: 0 0 24px rgba(61,220,151,.35)`.

### Typography
- **JetBrains Mono** (400–700): logo, page titles, section titles, numbers/XP/timers, code, terminal, tables' numeric cells, uppercase table headers (12px, letter-spacing .04em).
- **Inter** (400–600): body, buttons, nav, form labels.
- Sizes: page title 30px/700 mono; hero 34px/700 mono; card title 15–16px/600; body 14px; meta 12.5–13px.
- Every page title is preceded by a small mono "command" line, e.g. `$ ls ./tasks`, `$ cat progress.json`, `$ sort -rn scores | head`.

### Shape & spacing
- Radius: cards 14–18px, inputs/buttons 9–10px, chips 6px / pill 999px.
- Buttons/inputs: height 42–46px (touch target ≥44 for primary).
- Container: `max-width: 1320px; padding: 0 24px`. Section gap 24px.

### Background
`body { background: #0A0E14 url(/assets/wallpaper.svg) center top / cover fixed; }`
Cards are semi-transparent (`rgba(15,21,32,.85)`) so the wallpaper shows through; header uses `backdrop-filter: blur(12px)`.

---

## 2. Assets (`assets/`)
| File | Where |
|---|---|
| `wallpaper.svg` | Global page background (grid + dots + network nodes + soft glows) |
| `servers.svg` | Dashboard hero, Login left column, Profile card (faded) |
| `cover-nginx.svg` … `cover-cron.svg` | Task covers: task cards, learning path thumbnails, progress table, workspace header, admin task list |

Map task id → cover: `nginx-health→cover-nginx`, `docker-basics→cover-docker`, `systemd-service→cover-systemd`,
`user-management→cover-users`, `firewall-config→cover-firewall`, `cron-backup→cover-cron`.
Suggest adding an optional `cover`/`category`/`accent` field to `task.yaml` + `Task` type later; until then use a frontend map.

---

## 3. Shared components (build these first)
- `AppHeader` — logo (`devops_journey/lab`, green square with `>_` glyph), nav: Bosh sahifa · Vazifalar · Laboratoriya · Natijalar · Reyting; right side: ⌘K search button, live session pill (green dot + `mm:ss`, links to workspace, only when session active), avatar (links to profile). Active link: `bg surface-3`, inset border, green dot.
- `AppFooterStatus` — mono status bar: `● cluster: online · containers n/20 · image · version`.
- `Card`, `StatTile` (colored icon tile + label + mono value), `Badge` (done / active / open / locked variants), `SegmentedControl`, `ProgressSegments` (n equal segments), `TerminalWindow` (traffic-light title bar + mono body), `TaskCover`, `EmptyState`.
- Icons: keep `lucide-vue-next` (already installed). No emoji.

---

## 4. Pages → routes → existing code

| Design file | Route | Replaces / uses | API |
|---|---|---|---|
| `Login.dc.html` | `/login` | `views/LoginView.vue` | `POST /api/auth/login`, `/register` |
| `Main.dc.html` | `/` (dashboard) | new `DashboardView.vue` | `GET /api/tasks`, `/api/progress`, `/api/leaderboard` |
| `Tasks.dc.html` | `/tasks` | new `TasksView.vue` | `GET /api/tasks`, `/api/progress` |
| `Workspace.dc.html` | `/lab/:taskId?` | `views/LabView.vue`, `components/Terminal.vue`, `CheckResults.vue`, `TaskDetail.vue` | `POST/DELETE /api/sessions`, `POST /api/check/:id`, `ws /ws` |
| `Progress.dc.html` | `/progress` | replaces `ProgressSummary.vue` modal | `GET /api/progress` |
| `Leaderboard.dc.html` | `/leaderboard` | new | `GET /api/leaderboard` |
| `Profile.dc.html` | `/profile` | new (theme store moves here) | — (see §6) |
| `Admin.dc.html` | `/admin` | `views/AdminView.vue` | `/api/admin/stats, users, sessions, activity, resources, tasks/reload` |

### Page notes
- **Dashboard**: hero = greeting (`sardor@lab:~$ whoami`) + "continue" card for the in-progress task (4-segment check progress, "Terminalni ochish" primary, "Topshiriqni o'qish" secondary) + `servers.svg`. 4 stat tiles (total XP, completed, time in lab, rank). Learning path list with cover thumbnails + status badge. Right column: `tail -f activity.log` terminal card + quick links. Footer status bar.
- **Tasks**: filter bar (mono `>` search, status segmented: Barchasi/Yangi/Jarayonda/Yakunlangan, level: Har qanday/Oson/O'rta/Qiyin). Grid of cards (cover image, number chip, status badge on cover, title, category chip, level colored, `+xp`). Selected card → right detail panel: cover, `tasks/<id>`, description, 3 mini stats (xp, checks, image), "# nima tekshiriladi" list from `task.checks[].name`, CTA (Boshlash / Terminalga qaytish / Qayta ishlash) or locked notice.
- **Workspace** (full height, no main nav): breadcrumb `tasks / <id>`, 6-segment path indicator, session timer pill, "Qayta yaratish", "Tugatish" (danger outline). Left panel 380–470px: cover header with title, tabs Topshiriq / Maslahatlar / Natija. Hints unlock one at a time. Results: score `n/4 passed`, bar, per-check rows (green/red border, mono tip). "Keyingi vazifa →" when all pass. Right: terminal tab bar, xterm body, bottom bar with primary "Yechimni tekshirish". Bottom IDE status bar (green tint).
- **Progress**: conic ring %, XP, 4 mini stats; per-task table with cover thumbnails; skills bars per category (glowing); achievements grid (earned = colored, locked = dashed) with Hammasi/Olingan filter.
- **Leaderboard**: period segmented (hafta/oy/barcha vaqt — needs backend param), podium top-3 (gold `#F5B342`, silver `#C9D4E0`, bronze `#E0915A`), "your rank" banner, table with 6-cell progress.
- **Profile**: header card (avatar ring, username, level title, chips), side nav Profil / Ko'rinish / Bildirishnomalar / Xavfsizlik, terminal theme picker (Matrix/Ocean/Dracula/Amber CRT → xterm theme), font size, language.
- **Admin**: own header (`/admin`, `sudo` badge, "all systems operational"), side nav with counts. Overview: KPI tiles, 24h sessions bar chart, completion bars. Users table, Sessions table (`kill` button → terminate), Activity as `journalctl` terminal (INFO/WARN/ERROR colored), Resources as 20-cell meters, Tasks grid with covers + reload.

---

## 5. Behaviour
- Dark theme only by default (remove light/dark toggle or keep light as optional later). Remove old gradients/hover-scale effects.
- All data in the designs is sample data — wire to real API, show skeletons while loading and `EmptyState` when empty.
- Responsive: side panels stack under content on narrow screens; tables in `overflow-x-auto`.
- Accessibility: real `<button>`/`<a>`, `aria-current` for active nav, `aria-pressed` on segmented controls, `role="switch"` toggles, visible focus ring (`outline: 2px solid #3DDC97`).

## 6. Features that need backend work (mark as TODO if not implemented)
Achievements, skills by category, streak days, certificate PDF, leaderboard periods, notification settings,
profile fields (GitHub, goal), "recreate container", time-in-lab, password change, progress reset.
