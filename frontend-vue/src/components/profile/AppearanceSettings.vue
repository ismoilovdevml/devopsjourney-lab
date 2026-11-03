<!--
  AppearanceSettings — "Ko'rinish" section: terminal theme (radio group of live mini-terminal
  previews, arrow keys move the selection), terminal font size (13/14/16) with a full-size sample,
  and interface language (only O'zbekcha is available). Changes apply immediately and persist via the
  preferences store (localStorage), so there is no Save button.
  Props: none.
-->
<template>
  <section aria-labelledby="appearance-title" class="flex flex-col gap-6 rounded-panel border border-border bg-[rgba(15,21,32,0.88)] p-5 sm:p-6">
    <div>
      <h2 id="appearance-title" class="font-mono text-lg font-semibold text-text">Ko'rinish</h2>
      <p class="mt-0.5 text-[13px] text-muted">O'zgarishlar darhol qo'llanadi va shu brauzerda saqlanadi.</p>
    </div>

    <div class="flex flex-col gap-2.5">
      <span id="theme-label" class="font-medium text-text">Terminal mavzusi</span>
      <div
        role="radiogroup"
        aria-labelledby="theme-label"
        class="grid grid-cols-2 gap-2.5 lg:grid-cols-4"
        @keydown="onThemeKeydown"
      >
        <button
          v-for="t in themes"
          :key="t.id"
          :ref="(el) => setThemeRef(t.id, el)"
          type="button"
          role="radio"
          :aria-checked="t.id === prefs.terminalTheme"
          :tabindex="t.id === prefs.terminalTheme ? 0 : -1"
          :class="[
            'flex flex-col items-start gap-2.5 rounded-xl bg-surface-2 p-2.5 text-left font-medium text-text transition-colors',
            t.id === prefs.terminalTheme
              ? 'border-2 border-accent shadow-selected'
              : 'border border-border hover:border-border-stronger'
          ]"
          @click="prefs.setTerminalTheme(t.id)"
        >
          <span
            class="flex w-full flex-col justify-end gap-0.5 overflow-hidden rounded-lg border border-border px-2.5 pb-2 pt-2.5 font-mono text-[11px] leading-[1.45]"
            :style="{ background: t.meta.xterm.background, color: t.meta.xterm.foreground }"
            aria-hidden="true"
          >
            <span class="truncate"><span :style="{ color: t.meta.prompt }">$</span> whoami</span>
            <span class="truncate" :style="{ color: t.meta.xterm.green }">student</span>
            <span class="truncate"><span :style="{ color: t.meta.prompt }">$</span> ls <span :style="{ color: t.meta.xterm.blue }">/etc/nginx</span></span>
            <span class="flex items-center gap-1">
              <span :style="{ color: t.meta.prompt }">$</span>
              <span class="inline-block h-[11px] w-[6px]" :style="{ background: t.meta.xterm.cursor }" />
            </span>
          </span>
          <span class="flex w-full items-center justify-between gap-2 px-0.5">
            {{ t.meta.label }}
            <Check v-if="t.id === prefs.terminalTheme" class="h-4 w-4 text-accent" aria-hidden="true" />
          </span>
        </button>
      </div>
    </div>

    <div class="flex flex-col gap-2.5">
      <span id="font-label" class="font-medium text-text">Terminal shrifti</span>
      <SegmentedControl
        :model-value="prefs.fontSize"
        :options="sizeOptions"
        aria-label="Terminal shrifti o'lchami"
        class="self-start"
        @update:model-value="onFontSize"
      />
      <TerminalWindow title="namuna — bash" tone="terminal" :padded="false" class="mt-1">
        <div
          class="overflow-x-auto px-4 py-3.5 font-mono leading-[1.55]"
          :style="{ background: preview.xterm.background, color: preview.xterm.foreground, fontSize: `${prefs.fontSize}px` }"
          aria-label="Terminal ko'rinishi namunasi"
          role="img"
        >
          <div class="whitespace-pre"><span :style="{ color: preview.prompt }">student@lab:~$</span> systemctl is-active nginx</div>
          <div class="whitespace-pre" :style="{ color: preview.xterm.green }">active</div>
          <div class="whitespace-pre"><span :style="{ color: preview.prompt }">student@lab:~$</span> sudo ufw status</div>
          <div class="whitespace-pre">Status: <span :style="{ color: preview.xterm.green }">active</span></div>
          <div class="whitespace-pre"><span :style="{ color: preview.xterm.yellow }">22/tcp</span>   ALLOW   Anywhere</div>
          <div class="flex items-center gap-1.5 whitespace-pre">
            <span :style="{ color: preview.prompt }">student@lab:~$</span>
            <span class="inline-block w-[0.6em]" :style="{ height: '1.1em', background: preview.xterm.cursor }" aria-hidden="true" />
          </div>
        </div>
      </TerminalWindow>
    </div>

    <div class="flex max-w-[320px] flex-col">
      <label for="ui-language" class="label">Interfeys tili</label>
      <select id="ui-language" class="input h-[46px]" aria-describedby="ui-language-hint" :value="'uz'">
        <option value="uz">O'zbekcha</option>
        <option value="en" disabled>English — tez orada</option>
        <option value="ru" disabled>Русский — tez orada</option>
      </select>
      <p id="ui-language-hint" class="mt-1.5 text-xs text-muted">Boshqa tillar tez orada qo'shiladi.</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, type ComponentPublicInstance } from 'vue'
import { Check } from 'lucide-vue-next'
import SegmentedControl from '@/components/ui/SegmentedControl.vue'
import TerminalWindow from '@/components/ui/TerminalWindow.vue'
import {
  FONT_SIZES,
  TERMINAL_THEMES,
  usePreferencesStore,
  type TerminalFontSize,
  type TerminalThemeId
} from '@/stores/preferences'

const prefs = usePreferencesStore()

const themes = (Object.keys(TERMINAL_THEMES) as TerminalThemeId[]).map((id) => ({ id, meta: TERMINAL_THEMES[id] }))
const sizeOptions = FONT_SIZES.map((s) => ({ value: s, label: `${s} px` }))
const preview = computed(() => prefs.themeMeta)

const themeRefs = new Map<TerminalThemeId, HTMLElement>()
function setThemeRef(id: TerminalThemeId, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLElement) themeRefs.set(id, el)
  else themeRefs.delete(id)
}

function onFontSize(value: string | number) {
  const n = Number(value)
  if ((FONT_SIZES as readonly number[]).includes(n)) prefs.setFontSize(n as TerminalFontSize)
}

/** Radio-group keyboard model: arrows move and select, Home/End jump to the ends. */
function onThemeKeydown(e: KeyboardEvent) {
  const ids = themes.map((t) => t.id)
  const i = ids.indexOf(prefs.terminalTheme)
  let next = i
  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') next = (i + 1) % ids.length
  else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') next = (i - 1 + ids.length) % ids.length
  else if (e.key === 'Home') next = 0
  else if (e.key === 'End') next = ids.length - 1
  else return
  e.preventDefault()
  prefs.setTerminalTheme(ids[next])
  themeRefs.get(ids[next])?.focus()
}
</script>
