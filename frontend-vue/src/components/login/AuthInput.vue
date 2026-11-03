<!--
  AuthInput — labelled input for the auth form with inline hint/error wiring.
  Props:
    id: string                 (input id; hint/error ids derive from it)
    label: string
    modelValue: string         (v-model)
    type?: 'text' | 'email' | 'password'  (default 'text'; 'password' adds show/hide + caps-lock warning)
    autocomplete: string
    placeholder?: string
    error?: string             (sets aria-invalid and is announced via aria-describedby)
    hint?: string              (helper text, hidden while an error is shown)
    mono?: boolean             (JetBrains Mono value text; default true)
  Slots: label-extra (right side of the label row, e.g. "Unutdingizmi?").
  Emits: update:modelValue(string), blur(FocusEvent). Exposes focus().
-->
<template>
  <div class="flex flex-col gap-1.5">
    <div class="flex items-baseline justify-between gap-3">
      <label :for="id" class="text-[13px] font-medium text-text-2">{{ label }}</label>
      <slot name="label-extra" />
    </div>
    <div class="relative">
      <input
        :id="id"
        ref="inputEl"
        :value="modelValue"
        :type="inputType"
        :name="id"
        :autocomplete="autocomplete"
        :placeholder="placeholder"
        :aria-invalid="error ? 'true' : 'false'"
        :aria-describedby="describedBy"
        autocapitalize="none"
        :spellcheck="false"
        autocorrect="off"
        :class="[
          'input h-[46px] rounded-btn px-3.5 text-sm placeholder:text-faint',
          mono ? 'font-mono' : '',
          type === 'password' ? 'pr-12' : ''
        ]"
        @input="onInput"
        @blur="onBlur"
        @keydown="detectCapsLock"
        @keyup="detectCapsLock"
      />
      <button
        v-if="type === 'password'"
        type="button"
        class="absolute right-1.5 top-1/2 flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-ctl text-muted transition-colors hover:bg-surface-3 hover:text-text"
        :aria-label="revealed ? 'Parolni yashirish' : 'Parolni ko‘rsatish'"
        :aria-pressed="revealed"
        :aria-controls="id"
        @click="revealed = !revealed"
      >
        <EyeOff v-if="revealed" class="h-4 w-4" aria-hidden="true" />
        <Eye v-else class="h-4 w-4" aria-hidden="true" />
      </button>
    </div>
    <p v-if="error" :id="errorId" class="flex items-start gap-1.5 text-meta text-danger">
      <AlertCircle class="mt-[2px] h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <span>{{ error }}</span>
    </p>
    <p v-else-if="hint" :id="hintId" class="text-meta text-muted">{{ hint }}</p>
    <p
      v-if="type === 'password' && capsLock"
      :id="capsId"
      class="flex items-center gap-1.5 text-meta text-warn"
      role="status"
    >
      <ArrowBigUp class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      Caps Lock yoqilgan
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { AlertCircle, ArrowBigUp, Eye, EyeOff } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    id: string
    label: string
    modelValue: string
    type?: 'text' | 'email' | 'password'
    autocomplete: string
    placeholder?: string
    error?: string
    hint?: string
    mono?: boolean
  }>(),
  { type: 'text', placeholder: undefined, error: undefined, hint: undefined, mono: true }
)

const emit = defineEmits<{ 'update:modelValue': [value: string]; blur: [event: FocusEvent] }>()

const inputEl = ref<HTMLInputElement | null>(null)
const revealed = ref(false)
const capsLock = ref(false)

const errorId = computed(() => `${props.id}-error`)
const hintId = computed(() => `${props.id}-hint`)
const capsId = computed(() => `${props.id}-caps`)

const inputType = computed(() => (props.type === 'password' && revealed.value ? 'text' : props.type))

const describedBy = computed(() => {
  const ids: string[] = []
  if (props.error) ids.push(errorId.value)
  else if (props.hint) ids.push(hintId.value)
  if (props.type === 'password' && capsLock.value) ids.push(capsId.value)
  return ids.length ? ids.join(' ') : undefined
})

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}

function onBlur(event: FocusEvent) {
  capsLock.value = false
  emit('blur', event)
}

function detectCapsLock(event: KeyboardEvent) {
  if (props.type !== 'password' || typeof event.getModifierState !== 'function') return
  capsLock.value = event.getModifierState('CapsLock')
}

defineExpose({
  focus: () => inputEl.value?.focus()
})
</script>
