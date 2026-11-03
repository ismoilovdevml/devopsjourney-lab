/**
 * Lab workspace helpers: drive the real xterm.js terminal and the check flow through the UI.
 *
 * Terminal I/O:
 *   - input: click `.xterm` (focuses xterm's hidden textarea), then page.keyboard.type() + Enter.
 *   - output: xterm v5 uses its DOM renderer here, so the visible viewport is plain text in
 *     `.xterm-rows` (one <div> per row). Only the viewport is in the DOM (no scrollback), so
 *     helpers run `clear` when they need a clean screen.
 *   - completion: every command is followed by `echo "@@$((n+1)):$?@@"`. The typed line contains
 *     the arithmetic expression, the output contains the evaluated number, so the marker can only
 *     match real output (never the echoed input) and carries the exit code.
 */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { expect, type Locator, type Page } from '@playwright/test'

export const NGINX = { id: 'nginx-health', title: 'Nginx Web Server Setup', checks: 5 } as const
export const DOCKER = { id: 'docker-basics', title: 'Dockerfile & Compose Authoring' } as const

/** Generous timeout for anything that waits on a container (create, exec, check). */
export const LAB_TIMEOUT = 60_000

export function terminalRows(page: Page): Locator {
  return page.locator('.xterm-rows')
}

/** Visible terminal text, rows joined with '\n' and non-breaking spaces normalised. */
export async function terminalText(page: Page): Promise<string> {
  const rows = await terminalRows(page)
    .locator(':scope > div')
    .allTextContents()
    .catch(() => [] as string[])
  return rows.map((r) => r.replace(/ /g, ' ').trimEnd()).join('\n')
}

/** Lab status bar (bottom, green): "● ulangan" once the WebSocket is open. */
export function labStatusBar(page: Page): Locator {
  return page.locator('footer').filter({ hasText: /tasks\// })
}

/** Waits until the terminal WebSocket is open and the shell printed a prompt. */
export async function waitForTerminalReady(page: Page): Promise<void> {
  await expect(labStatusBar(page)).toContainText('ulangan', { timeout: LAB_TIMEOUT })
  await expect(page.locator('.xterm')).toBeVisible()
  await expect
    .poll(() => terminalText(page), { timeout: LAB_TIMEOUT, message: 'shell prompt in terminal' })
    .toMatch(/[#$]\s*$/m)
}

async function focusTerminal(page: Page): Promise<void> {
  await page.locator('.xterm').click()
  await expect(page.locator('.xterm-helper-textarea')).toBeFocused()
}

/** Types one line into the terminal and presses Enter (no completion wait). */
export async function typeLine(page: Page, line: string): Promise<void> {
  await focusTerminal(page)
  await page.keyboard.type(line)
  await page.keyboard.press('Enter')
}

export interface CommandResult {
  exitCode: number
  /** Visible terminal text when the command finished. */
  screen: string
}

let markerSeq = Math.floor(Math.random() * 100_000) * 10

/**
 * Runs `cmd` in the lab shell and waits for it to finish.
 * `clear: true` clears the screen first, so `screen` only holds this command's output.
 */
export async function runCommand(
  page: Page,
  cmd: string,
  opts: { clear?: boolean; timeout?: number } = {}
): Promise<CommandResult> {
  const n = (markerSeq += 7)
  const expected = new RegExp(`@@${n + 1}:(\\d+)@@`)
  const line = `${opts.clear ? 'clear; ' : ''}${cmd}; echo "@@$((${n}+1)):$?@@"`
  await typeLine(page, line)
  let screen = ''
  await expect
    .poll(
      async () => {
        screen = await terminalText(page)
        return expected.test(screen)
      },
      { timeout: opts.timeout ?? LAB_TIMEOUT, message: `terminal output of: ${cmd}` }
    )
    .toBe(true)
  return { exitCode: Number(expected.exec(screen)?.[1] ?? -1), screen }
}

/**
 * Lines of the reference solution (e2e/fixtures/solutions/<task>.sh) as single-line shell commands:
 * shebang, comments, blank lines and `set -euo pipefail` are dropped.
 */
export function solutionCommands(taskId: string): string[] {
  const src = readFileSync(fileURLToPath(new URL(`./solutions/${taskId}.sh`, import.meta.url)), 'utf8')
  return src
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('#') && !/^set\s+-/.test(l))
}

/** Types every solution command into the terminal; each must exit 0. */
export async function solveInTerminal(page: Page, taskId: string): Promise<void> {
  for (const cmd of solutionCommands(taskId)) {
    const res = await runCommand(page, cmd)
    expect(res.exitCode, `solution step failed: ${cmd}\n${res.screen}`).toBe(0)
  }
}

// ---------- Workspace UI ----------

export function resultsPanel(page: Page): Locator {
  return page.getByRole('tabpanel')
}

export function passedRows(page: Page): Locator {
  return resultsPanel(page).getByRole('img', { name: "O'tdi" })
}

export function failedRows(page: Page): Locator {
  return resultsPanel(page).getByRole('img', { name: "O'tmadi" })
}

export interface Score {
  score: number
  total: number
}

/** Reads "N/M passed" from the Natija tab. */
export async function readScore(page: Page): Promise<Score> {
  const text = (await resultsPanel(page).getByText(/\d+\/\d+ passed/).textContent()) ?? ''
  const m = /(\d+)\/(\d+) passed/.exec(text)
  expect(m, `score text: ${text}`).not.toBeNull()
  return { score: Number(m![1]), total: Number(m![2]) }
}

/**
 * Runs the check (button or Ctrl/Cmd+Enter) and waits for the POST /api/check response and the
 * rendered result. Returns the API body.
 */
export async function runCheck(
  page: Page,
  via: 'button' | 'shortcut' = 'button'
): Promise<{ passed: boolean; score: number; total: number }> {
  const response = page.waitForResponse(
    (r) => r.url().includes('/api/check/') && r.request().method() === 'POST',
    { timeout: LAB_TIMEOUT }
  )
  if (via === 'shortcut') {
    await focusTerminal(page)
    await page.keyboard.press('ControlOrMeta+Enter')
  } else {
    await page.getByRole('region', { name: 'Terminal' }).getByRole('button', { name: 'Yechimni tekshirish' }).click()
  }
  const res = await response
  expect(res.status(), await res.text()).toBe(200)
  const body = (await res.json()) as { passed: boolean; score: number; total: number }
  await expect(page.getByRole('tab', { name: /Natija/ })).toHaveAttribute('aria-selected', 'true')
  await expect(resultsPanel(page).getByText(`${body.score}/${body.total} passed`)).toBeVisible()
  return body
}

/** From /tasks: selects the task card and clicks "Laboratoriyani boshlash"; waits for the lab terminal. */
export async function startLabFromTasks(page: Page, task: { id: string; title: string }): Promise<void> {
  await page.goto(`/tasks?task=${task.id}`)
  const detail = page.getByRole('complementary', { name: task.title })
  await detail.getByRole('button', { name: 'Laboratoriyani boshlash' }).click()
  await expect(page).toHaveURL(new RegExp(`/lab/${task.id}$`), { timeout: LAB_TIMEOUT })
  await waitForTerminalReady(page)
}

/** Opens /lab/<id> and starts the environment from the lab start screen. */
export async function startLabInWorkspace(page: Page, taskId: string): Promise<void> {
  await page.goto(`/lab/${taskId}`)
  await page.getByRole('button', { name: 'Muhitni ishga tushirish' }).click()
  await waitForTerminalReady(page)
}

/** Full UI solve of task 1: start the lab, type the reference solution, check until 5/5. */
export async function completeNginxViaUi(page: Page): Promise<void> {
  await startLabFromTasks(page, NGINX)
  await solveInTerminal(page, NGINX.id)
  const res = await runCheck(page)
  expect(res, 'nginx-health check after solving').toMatchObject({ passed: true, score: NGINX.checks, total: NGINX.checks })
}

/** Confirm button of the open modal dialog. */
export function dialogButton(page: Page, name: string): Locator {
  return page.getByRole('dialog').getByRole('button', { name, exact: true })
}
