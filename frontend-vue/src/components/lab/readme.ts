/**
 * Splits a task README (markdown) for the lab workspace.
 *
 * Task READMEs (backend-go/tasks/<id>/README.md) start with a `# Title` and contain a `## Hints`
 * section with near-complete solutions. In the workspace the title is already shown in the panel
 * header, and the solution section is moved behind the hint unlock flow so the "Topshiriq" tab
 * does not give the answer away.
 *
 * Fenced code blocks are respected (a `# comment` inside ``` is not a heading).
 */

export interface SplitReadme {
  /** README without the leading H1 and without the hints section. */
  body: string
  /** Content of the `## Hints` / `## Maslahatlar` section (without its heading), or '' if absent. */
  hints: string
}

const HINTS_HEADING = /^##\s+(hints?|maslahat(lar)?|solution|yechim)\s*$/i
const FENCE = /^\s*(```|~~~)/

export function splitReadme(markdown: string | null | undefined): SplitReadme {
  if (!markdown) return { body: '', hints: '' }

  const body: string[] = []
  const hints: string[] = []
  let inFence = false
  let inHints = false
  let titleSkipped = false

  for (const line of markdown.replace(/\r\n?/g, '\n').split('\n')) {
    if (FENCE.test(line)) {
      inFence = !inFence
    } else if (!inFence && /^#\s/.test(line)) {
      // Leading H1 = task title (already in the panel header).
      if (!titleSkipped && body.every((l) => !l.trim())) {
        titleSkipped = true
        continue
      }
      inHints = false
    } else if (!inFence && /^##\s/.test(line)) {
      inHints = HINTS_HEADING.test(line.trim())
      if (inHints) continue
    }

    if (inHints) hints.push(line)
    else body.push(line)
  }

  return { body: body.join('\n').trim(), hints: hints.join('\n').trim() }
}
