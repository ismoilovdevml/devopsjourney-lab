/** Shared types for the lab workspace components. */

/** Mirrors the `status` events emitted by components/Terminal.vue, plus 'idle' (no session). */
export type ConnectionState = 'idle' | 'connecting' | 'open' | 'reconnecting' | 'closed' | 'failed'

export type LabTab = 'task' | 'hints' | 'results'

export const CONNECTION_LABEL: Record<ConnectionState, string> = {
  idle: "sessiya yo'q",
  connecting: 'ulanmoqda…',
  open: 'ulangan',
  reconnecting: 'qayta ulanmoqda…',
  closed: 'uzilgan',
  failed: "ulanib bo'lmadi"
}

/** Tailwind text color class for a connection state dot. */
export const CONNECTION_COLOR: Record<ConnectionState, string> = {
  idle: 'text-faint',
  connecting: 'text-warn',
  open: 'text-accent',
  reconnecting: 'text-warn',
  closed: 'text-danger',
  failed: 'text-danger'
}
