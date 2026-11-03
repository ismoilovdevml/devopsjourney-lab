/**
 * Per-browser UI preferences, persisted to localStorage (key `dj.prefs`). No secrets here.
 *
 * State:   terminalTheme ('matrix' | 'ocean' | 'dracula' | 'amber'), fontSize (13 | 14 | 16)
 * Getters: xtermTheme (xterm ITheme for the current choice), themeMeta (label + swatch colors)
 * Actions: setTerminalTheme(id), setFontSize(px)
 * Constants: TERMINAL_THEMES (id -> { label, background, prompt, xterm }), FONT_SIZES
 */
import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import type { ITheme } from 'xterm'

export type TerminalThemeId = 'matrix' | 'ocean' | 'dracula' | 'amber'
export type TerminalFontSize = 13 | 14 | 16

export interface TerminalThemeMeta {
  label: string
  /** Swatch background (also the xterm background). */
  background: string
  /** Swatch prompt color. */
  prompt: string
  xterm: ITheme
}

export const FONT_SIZES: readonly TerminalFontSize[] = [13, 14, 16]

export const TERMINAL_THEMES: Record<TerminalThemeId, TerminalThemeMeta> = {
  matrix: {
    label: 'Matrix',
    background: '#070A0F',
    prompt: '#3DDC97',
    xterm: {
      background: '#070A0F',
      foreground: '#C9F5DF',
      cursor: '#3DDC97',
      cursorAccent: '#070A0F',
      selectionBackground: 'rgba(61,220,151,0.30)',
      black: '#0B1018',
      red: '#F87171',
      green: '#3DDC97',
      yellow: '#F5B342',
      blue: '#38BDF8',
      magenta: '#A78BFA',
      cyan: '#2DD4BF',
      white: '#C9D4E0',
      brightBlack: '#64748B',
      brightRed: '#FCA5A5',
      brightGreen: '#7EF0BE',
      brightYellow: '#FCD34D',
      brightBlue: '#7DD3FC',
      brightMagenta: '#C4B5FD',
      brightCyan: '#5EEAD4',
      brightWhite: '#F1F5F9'
    }
  },
  ocean: {
    label: 'Ocean',
    background: '#0B1626',
    prompt: '#38BDF8',
    xterm: {
      background: '#0B1626',
      foreground: '#D6E6F5',
      cursor: '#38BDF8',
      cursorAccent: '#0B1626',
      selectionBackground: 'rgba(56,189,248,0.30)',
      black: '#0F1E33',
      red: '#F87171',
      green: '#4ADE80',
      yellow: '#FBBF24',
      blue: '#38BDF8',
      magenta: '#A78BFA',
      cyan: '#22D3EE',
      white: '#CBD5E1',
      brightBlack: '#64748B',
      brightRed: '#FCA5A5',
      brightGreen: '#86EFAC',
      brightYellow: '#FDE68A',
      brightBlue: '#7DD3FC',
      brightMagenta: '#C4B5FD',
      brightCyan: '#67E8F9',
      brightWhite: '#F8FAFC'
    }
  },
  dracula: {
    label: 'Dracula',
    background: '#1A1526',
    prompt: '#A78BFA',
    xterm: {
      background: '#1A1526',
      foreground: '#F8F8F2',
      cursor: '#A78BFA',
      cursorAccent: '#1A1526',
      selectionBackground: 'rgba(167,139,250,0.30)',
      black: '#21222C',
      red: '#FF5555',
      green: '#50FA7B',
      yellow: '#F1FA8C',
      blue: '#BD93F9',
      magenta: '#FF79C6',
      cyan: '#8BE9FD',
      white: '#F8F8F2',
      brightBlack: '#6272A4',
      brightRed: '#FF6E6E',
      brightGreen: '#69FF94',
      brightYellow: '#FFFFA5',
      brightBlue: '#D6ACFF',
      brightMagenta: '#FF92DF',
      brightCyan: '#A4FFFF',
      brightWhite: '#FFFFFF'
    }
  },
  amber: {
    label: 'Amber CRT',
    background: '#140F06',
    prompt: '#F5B342',
    xterm: {
      background: '#140F06',
      foreground: '#F5B342',
      cursor: '#FFD27A',
      cursorAccent: '#140F06',
      selectionBackground: 'rgba(245,179,66,0.30)',
      black: '#1F170A',
      red: '#F28B4B',
      green: '#E8B04A',
      yellow: '#FFD27A',
      blue: '#C98A2E',
      magenta: '#E0915A',
      cyan: '#F0C060',
      white: '#F5D9A0',
      brightBlack: '#7A5A22',
      brightRed: '#FFA66B',
      brightGreen: '#FFC966',
      brightYellow: '#FFE2A3',
      brightBlue: '#E0A94A',
      brightMagenta: '#F2AE7C',
      brightCyan: '#FFD98A',
      brightWhite: '#FFF1D0'
    }
  }
}

const STORAGE_KEY = 'dj.prefs'

interface StoredPrefs {
  terminalTheme?: string
  fontSize?: number
}

function isThemeId(v: unknown): v is TerminalThemeId {
  return typeof v === 'string' && v in TERMINAL_THEMES
}

function isFontSize(v: unknown): v is TerminalFontSize {
  return typeof v === 'number' && (FONT_SIZES as readonly number[]).includes(v)
}

function readStored(): StoredPrefs {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed: unknown = raw ? JSON.parse(raw) : null
    return parsed && typeof parsed === 'object' ? (parsed as StoredPrefs) : {}
  } catch {
    return {}
  }
}

export const usePreferencesStore = defineStore('preferences', () => {
  const stored = readStored()
  const terminalTheme = ref<TerminalThemeId>(isThemeId(stored.terminalTheme) ? stored.terminalTheme : 'matrix')
  const fontSize = ref<TerminalFontSize>(isFontSize(stored.fontSize) ? stored.fontSize : 14)

  const themeMeta = computed(() => TERMINAL_THEMES[terminalTheme.value])
  const xtermTheme = computed<ITheme>(() => themeMeta.value.xterm)

  function setTerminalTheme(id: TerminalThemeId): void {
    terminalTheme.value = id
  }

  function setFontSize(size: TerminalFontSize): void {
    fontSize.value = size
  }

  watch([terminalTheme, fontSize], ([theme, size]) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ terminalTheme: theme, fontSize: size }))
    } catch {
      // Storage may be unavailable (private mode); preferences then last for this tab only.
    }
  })

  return { terminalTheme, fontSize, themeMeta, xtermTheme, setTerminalTheme, setFontSize }
})
