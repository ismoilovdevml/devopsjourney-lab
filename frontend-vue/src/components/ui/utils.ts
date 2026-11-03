/** `#RRGGBB` + alpha -> `rgba(r,g,b,a)`. Non-hex input is returned unchanged. */
export function tint(hex: string, alpha: number): string {
  const m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex.trim())
  if (!m) return hex
  const [r, g, b] = [m[1], m[2], m[3]].map((h) => parseInt(h, 16))
  return `rgba(${r},${g},${b},${alpha})`
}

let counter = 0
/** Unique DOM id for aria wiring (client-only app, so a counter is enough). */
export function uid(prefix = 'ui'): string {
  counter += 1
  return `${prefix}-${counter}`
}
