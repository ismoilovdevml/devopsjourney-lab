# Prompt for Claude Code

Copy this folder into the repo as `design/` (so the path is `devopsjourney-lab/design/`), then paste the prompt below into Claude Code.

---

Implement the new UI for `frontend-vue/` from the design handoff in `design/`.

Read first:
- `design/DESIGN.md` — tokens, components, page → route → API mapping, behaviour.
- `design/*.dc.html` — visual reference for each page (layout, copy, colors are inline; the `<script>` blocks hold sample data and interaction logic — reimplement in Vue, do not copy the runtime).
- `design/assets/*.svg` — copy to `frontend-vue/public/assets/` and use as-is.

Work plan:
1. Tokens: extend `frontend-vue/tailwind.config.js` with the colors from DESIGN.md §1 (bg, surface, border, text, muted, accent, info, warn, danger, violet, rose, teal), fonts `mono: JetBrains Mono`, `sans: Inter` (load from Google Fonts in `index.html`). Set the global background + wallpaper in `src/assets/main.css`. Remove old gradient/hover-scale styles.
2. Shared components in `src/components/ui/`: AppHeader, AppFooterStatus, Card, StatTile, Badge, SegmentedControl, ProgressSegments, TerminalWindow, TaskCover, EmptyState. Add `src/constants/tasks.ts` mapping task id → cover/accent/category.
3. Router (`src/router.ts`): `/login`, `/` dashboard, `/tasks`, `/lab/:taskId?`, `/progress`, `/leaderboard`, `/profile`, `/admin` (keep the existing auth/admin guards).
4. Pages, one at a time, matching the design: LoginView, DashboardView, TasksView, LabView (reuse Terminal.vue / xterm and session store; restyle xterm with the dark theme), ProgressView, LeaderboardView, ProfileView, AdminView.
5. Use real data from `src/services/api.ts` and the Pinia stores. For features without backend support (DESIGN.md §6) render the UI with a clear TODO / disabled state — do not invent data.
6. UI copy in Uzbek (Latin) as in the designs; task titles stay as in `task.yaml`.
7. Run `npm run build` and `npm run lint` in `frontend-vue/` and fix errors. Check each page at 1440px and 390px width.

Commit in small steps (tokens → components → each page).
