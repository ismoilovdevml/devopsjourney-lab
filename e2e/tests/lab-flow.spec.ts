/**
 * Lab workspace end-to-end: real container, real terminal (xterm over /ws), real checks.
 * Every test runs as a fresh student; the `student` fixture stops the active session on teardown.
 */
import { test, expect } from '../fixtures'
import {
  DOCKER,
  LAB_TIMEOUT,
  NGINX,
  dialogButton,
  failedRows,
  labStatusBar,
  passedRows,
  readScore,
  runCheck,
  runCommand,
  solveInTerminal,
  startLabFromTasks,
  waitForTerminalReady
} from '../fixtures/lab'

test.describe('lab workspace', () => {
  test('core journey: start task 1, use the terminal, fail, solve, pass, unlock task 2', async ({
    page,
    student,
    studentApi
  }) => {
    test.setTimeout(240_000)

    // 1. Start from /tasks.
    await startLabFromTasks(page, NGINX)
    await expect(page.getByRole('heading', { level: 1, name: NGINX.title })).toBeVisible()
    await expect(labStatusBar(page)).toContainText(`tasks/${NGINX.id}`)

    // 2. The terminal is a real shell.
    const token = `e2e-${Date.now().toString(36)}`
    const echo = await runCommand(page, `echo ${token}-$((6*7))-ok`)
    expect(echo.exitCode).toBe(0)
    expect(echo.screen).toContain(`${token}-42-ok`)

    // 3. Checking before solving fails with red rows.
    const before = await runCheck(page, 'button')
    expect(before.passed).toBe(false)
    expect(before.total).toBe(NGINX.checks)
    expect(before.score).toBeLessThan(before.total)
    expect(await readScore(page)).toEqual({ score: before.score, total: before.total })
    await expect(failedRows(page)).toHaveCount(before.total - before.score)
    await expect(failedRows(page).first()).toBeVisible()
    await expect(page.getByRole('button', { name: /Keyingi vazifa/ })).toHaveCount(0)
    await expect(page.getByRole('tab', { name: /Natija/ })).toContainText(`${before.score}/${before.total}`)

    // 4. Solve by typing the reference solution, then check with the keyboard shortcut.
    await solveInTerminal(page, NGINX.id)
    const after = await runCheck(page, 'shortcut')
    expect(after).toMatchObject({ passed: true, score: NGINX.checks, total: NGINX.checks })
    await expect(passedRows(page)).toHaveCount(NGINX.checks)
    await expect(failedRows(page)).toHaveCount(0)
    await expect(page.getByRole('button', { name: /Keyingi vazifa/ })).toBeVisible()

    // 5. /tasks: task 1 done, task 2 unlocked.
    await page.getByRole('navigation', { name: "Yo'l" }).getByRole('link', { name: 'tasks' }).click()
    await expect(page).toHaveURL(/\/tasks/)
    const list = page.getByRole('region', { name: "Vazifalar ro'yxati" })
    await expect(list.getByRole('button', { name: new RegExp(NGINX.title) })).toContainText('Yakunlangan')
    const dockerCard = list.getByRole('button', { name: new RegExp(DOCKER.title) })
    await expect(dockerCard).not.toContainText('Qulflangan')
    await expect(dockerCard).toContainText('Yangi')
    await dockerCard.click()
    const dockerDetail = page.getByRole('complementary', { name: DOCKER.title })
    await expect(dockerDetail.getByRole('button', { name: 'Laboratoriyani boshlash' })).toBeVisible()
    await expect(dockerDetail.getByText('Ochish uchun avvalgi vazifalarni yakunlang')).toHaveCount(0)

    // 6. /progress: 1 completed, XP > 0.
    await page.goto('/progress')
    const overview = page.getByRole('region', { name: "Umumiy ko'rsatkichlar" })
    await expect(overview).toContainText(/Yakunlangan\s*1\/6/)
    const xpText = (await overview.getByText(/\d+ \/ \d+ xp/).textContent()) ?? ''
    expect(Number(/(\d+) \//.exec(xpText)?.[1])).toBeGreaterThan(0)

    // 7. Dashboard stat tiles.
    await page.goto('/')
    const stats = page.getByRole('region', { name: 'Statistika' })
    await expect(stats).toContainText(/Yakunlangan\s*1\s*\/\s*6/)
    await expect(stats).toContainText(/Umumiy ball\s*100\s*\/\s*\d+/)
    await expect(stats).toContainText(/Reyting\s*#\d+/)

    // 8. Leaderboard lists the student with a positive score.
    await page.goto('/leaderboard')
    const meRow = page.locator('tr[aria-current="true"]')
    await expect(meRow).toContainText(student.username)
    const score = Number((await meRow.locator('td').last().textContent())?.trim())
    expect(score).toBeGreaterThan(0)

    // API agrees with the UI.
    const progress = await (await studentApi.get('/api/progress')).json()
    expect(progress.completed).toBe(1)
  })

  test('hints unlock one at a time', async ({ page, student }) => {
    void student
    await page.goto(`/lab/${NGINX.id}`)
    await page.getByRole('tab', { name: 'Maslahatlar' }).click()
    const panel = page.getByRole('tabpanel')
    const open = panel.getByRole('button', { name: 'Ochish' })

    // 3 task.yaml hints + the README "Hints" section as the final, detailed hint.
    await expect(panel).toContainText('0 / 4 ochildi')
    await expect(open).toHaveCount(4)
    await expect(open.nth(0)).toBeEnabled()
    for (let i = 1; i < 4; i++) await expect(open.nth(i)).toBeDisabled()
    await expect(panel.getByRole('region')).toHaveCount(0)

    await open.nth(0).click()
    await expect(panel).toContainText('1 / 4 ochildi')
    await expect(panel.getByRole('region', { name: '1-maslahat' })).toContainText('/var/www/html')
    await expect(open).toHaveCount(3)
    await expect(open.nth(0)).toBeEnabled()
    await expect(open.nth(1)).toBeDisabled()

    await open.nth(0).click()
    await expect(panel).toContainText('2 / 4 ochildi')
    await expect(panel.getByRole('region', { name: '2-maslahat' })).toContainText('echo ok > /var/www/html/index.html')
    await expect(panel.getByRole('region', { name: '3-maslahat' })).toHaveCount(0)

    // Unlocked count survives a reload (sessionStorage).
    await page.reload()
    await page.getByRole('tab', { name: 'Maslahatlar' }).click()
    await expect(page.getByRole('tabpanel')).toContainText('2 / 4 ochildi')
  })

  test('"Qayta yaratish" gives a fresh container', async ({ page, studentApi }) => {
    test.setTimeout(180_000)
    await startLabFromTasks(page, NGINX)
    const { sessionId: oldId } = await (await studentApi.get('/api/sessions/active')).json()

    expect((await runCommand(page, 'touch /root/marker && ls /root/marker')).exitCode).toBe(0)

    await page.getByRole('banner').getByRole('button', { name: 'Qayta yaratish' }).click()
    await expect(page.getByRole('dialog', { name: 'Muhitni qayta yaratish' })).toBeVisible()
    const recreated = page.waitForResponse(
      (r) => /\/api\/sessions\/[^/]+\/recreate$/.test(r.url()) && r.request().method() === 'POST',
      { timeout: LAB_TIMEOUT }
    )
    await dialogButton(page, 'Qayta yaratish').click()
    expect((await recreated).status()).toBe(201)
    await expect(page.getByRole('dialog')).toHaveCount(0)

    const { sessionId: newId } = await (await studentApi.get('/api/sessions/active')).json()
    expect(newId).not.toBe(oldId)

    await waitForTerminalReady(page)
    const ls = await runCommand(page, 'ls /root/marker', { clear: true })
    expect(ls.exitCode).not.toBe(0)
    expect(ls.screen).toContain('No such file or directory')
  })

  test('"Tugatish" stops the session and returns to /tasks', async ({ page, studentApi }) => {
    await startLabFromTasks(page, NGINX)
    await page.getByRole('banner').getByRole('button', { name: 'Tugatish' }).click()
    await expect(page.getByRole('dialog', { name: 'Sessiyani tugatish' })).toBeVisible()
    await dialogButton(page, 'Tugatish').click()

    await expect(page).toHaveURL(/\/tasks$/, { timeout: LAB_TIMEOUT })
    expect((await studentApi.get('/api/sessions/active')).status()).toBe(204)
    // No session pill in the header any more.
    await expect(page.getByRole('link', { name: /Faol sessiya/ })).toHaveCount(0)
  })

  test('reloading /lab/<id> reconnects the terminal to the same session', async ({ page, studentApi }) => {
    await startLabFromTasks(page, NGINX)
    const { sessionId } = await (await studentApi.get('/api/sessions/active')).json()
    expect((await runCommand(page, 'echo persisted > /tmp/e2e-reload')).exitCode).toBe(0)

    await page.reload()
    await waitForTerminalReady(page)
    const res = await runCommand(page, 'cat /tmp/e2e-reload', { clear: true })
    expect(res.exitCode).toBe(0)
    expect(res.screen).toMatch(/^persisted$/m)
    expect((await (await studentApi.get('/api/sessions/active')).json()).sessionId).toBe(sessionId)
  })

  test('header session pill shows a mm:ss countdown and links to the lab', async ({ page, student }) => {
    void student
    await startLabFromTasks(page, NGINX)
    await expect(page.getByRole('timer')).toContainText(/sessiya \d{2}:\d{2}/)

    await page.goto('/tasks')
    const pill = page.getByRole('banner').getByRole('link', { name: /Faol sessiya, .* qoldi/ })
    await expect(pill).toBeVisible()
    await expect(pill).toHaveText(/^\d{2}:\d{2}$/)
    const first = (await pill.textContent())?.trim()
    await expect.poll(async () => (await pill.textContent())?.trim(), { timeout: 5_000 }).not.toBe(first)

    await pill.click()
    await expect(page).toHaveURL(new RegExp(`/lab/${NGINX.id}$`))
    await waitForTerminalReady(page)
  })

  test('checks ignore a root-controlled ~/.bash_profile (cheat resistance)', async ({ page, student }) => {
    void student
    // Known bug H1: checks run as `bash -lc` (backend-go/internal/docker/exec.go), so a root-written
    // ~/.bash_profile containing `exit 0` makes every check exit 0. Remove this line once fixed.
    test.fail(true, 'BUG H1: checks honour root-controlled bash_profile')
    await startLabFromTasks(page, NGINX)
    expect((await runCommand(page, `echo 'exit 0' > /root/.bash_profile`)).exitCode).toBe(0)

    const res = await runCheck(page)
    expect(res.passed, 'an unsolved task must not pass because of ~/.bash_profile').toBe(false)
    expect(res.score).toBeLessThan(res.total)
  })
})
