/**
 * Markdown -> sanitized HTML for task READMEs (safe to bind with v-html).
 *
 * - Parsed with `marked` (GFM, synchronous).
 * - Sanitized with a private DOMPurify instance (scripts, event handlers, javascript: URLs, iframes,
 *   forms and inline styles are stripped).
 * - Every link that leaves the page gets `target="_blank" rel="noopener noreferrer"`;
 *   in-page anchors (`#section`) are left alone.
 *
 * Usage: <div class="prose-readme" v-html="renderMarkdown(task.readme)" />
 */
import DOMPurify from 'dompurify'
import { Marked } from 'marked'

const marked = new Marked({ gfm: true, breaks: false, async: false })

const purifier = DOMPurify(window)

purifier.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName !== 'A') return
  const href = node.getAttribute('href')
  if (!href || href.startsWith('#')) return
  node.setAttribute('target', '_blank')
  node.setAttribute('rel', 'noopener noreferrer')
})

const SANITIZE_OPTIONS = {
  USE_PROFILES: { html: true },
  FORBID_TAGS: ['style', 'iframe', 'form', 'input', 'button', 'textarea', 'select', 'object', 'embed'],
  FORBID_ATTR: ['style']
}

/** Render untrusted markdown to sanitized HTML. Returns '' for empty input. */
export function renderMarkdown(md: string): string {
  if (!md || !md.trim()) return ''
  const html = marked.parse(md, { async: false }) as string
  return purifier.sanitize(html, SANITIZE_OPTIONS)
}

export default renderMarkdown
