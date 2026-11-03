<!--
  AuthCard — "ssh student@devops-lab" window with Kirish / Ro'yxatdan o'tish tabs and the auth form.
  Props:
    mode: AuthMode            (v-model:mode; 'login' | 'register')
    loading?: boolean         (disables submit and shows a spinner)
    serverError?: AuthErrorView | null  (result of the last failed request; field messages win over form)
  Emits:
    update:mode(AuthMode)
    submit({ mode, username, email, password })  — values already validated and normalized
    clear-server-error()      (user edited something after a failed request)
-->
<template>
  <div
    class="w-full max-w-[420px] overflow-hidden rounded-hero border border-border-strong bg-[rgba(13,18,27,0.92)] shadow-[0_30px_80px_rgba(0,0,0,0.5)] backdrop-blur-[10px]"
  >
    <div class="flex items-center gap-2 border-b border-border bg-titlebar px-4 py-3" aria-hidden="true">
      <span class="h-2.5 w-2.5 rounded-full bg-danger" />
      <span class="h-2.5 w-2.5 rounded-full bg-warn" />
      <span class="h-2.5 w-2.5 rounded-full bg-accent" />
      <span class="ml-2 font-mono text-meta text-muted">ssh student@devops-lab</span>
    </div>

    <div class="flex flex-col gap-[22px] p-5 sm:p-7">
      <div role="tablist" aria-label="Kirish turi" class="flex gap-1 rounded-btn border border-border bg-surface-2 p-1">
        <button
          v-for="tab in TABS"
          :id="`auth-tab-${tab.mode}`"
          :key="tab.mode"
          :ref="(el) => setTabRef(tab.mode, el)"
          type="button"
          role="tab"
          :aria-selected="mode === tab.mode"
          aria-controls="auth-panel"
          :tabindex="mode === tab.mode ? 0 : -1"
          :class="[
            'h-9 flex-1 rounded-[7px] font-medium transition-colors',
            mode === tab.mode ? 'bg-raised text-text shadow-ring-strong' : 'bg-transparent text-muted hover:text-text'
          ]"
          @click="setMode(tab.mode)"
          @keydown="onTabKeydown"
        >
          {{ tab.label }}
        </button>
      </div>

      <div id="auth-panel" role="tabpanel" :aria-labelledby="`auth-tab-${mode}`" class="flex flex-col gap-[22px]">
        <div>
          <h2 id="auth-heading" class="text-2xl font-semibold tracking-[-0.01em]">{{ copy.heading }}</h2>
          <p class="mt-1 text-muted">{{ copy.sub }}</p>
        </div>

        <form class="flex flex-col gap-4" novalidate aria-labelledby="auth-heading" @submit.prevent="onSubmit">
          <AuthInput
            id="auth-username"
            ref="usernameRef"
            v-model="values.username"
            label="Foydalanuvchi nomi"
            autocomplete="username"
            placeholder="sardor"
            :error="fieldError('username')"
            :hint="mode === 'register' ? 'Kichik lotin harflari, raqamlar, “_” yoki “-”' : undefined"
            @blur="onUsernameBlur"
          />

          <AuthInput
            v-if="mode === 'register'"
            id="auth-email"
            ref="emailRef"
            v-model="values.email"
            type="email"
            label="Email"
            autocomplete="email"
            placeholder="siz@example.com"
            :error="fieldError('email')"
            @blur="onEmailBlur"
          />

          <AuthInput
            id="auth-password"
            ref="passwordRef"
            v-model="values.password"
            type="password"
            label="Parol"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            placeholder="••••••••"
            :error="fieldError('password')"
            :hint="mode === 'register' ? 'Kamida 8 ta belgi' : undefined"
            @blur="touch('password')"
          >
            <template v-if="mode === 'login'" #label-extra>
              <span class="text-[13px] text-muted" title="Parolni tiklash tez orada qo‘shiladi">
                Unutdingizmi? <span class="font-mono text-xs text-faint">· tez orada</span>
              </span>
            </template>
          </AuthInput>

          <div
            v-if="formError"
            role="alert"
            class="flex items-start gap-2 rounded-btn border border-danger/35 bg-danger/10 px-3 py-2.5 text-[13px] text-danger"
          >
            <AlertTriangle class="mt-[2px] h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{{ formError }}</span>
          </div>

          <Button type="submit" variant="primary" size="lg" block :loading="loading" :icon-right="ArrowRight" class="mt-1">
            {{ copy.cta }}
          </Button>
        </form>

        <p class="text-center text-[13px] text-muted">
          {{ copy.switchText }}
          <button
            type="button"
            class="rounded-sm text-accent underline underline-offset-2 hover:text-accent-hover"
            @click="setMode(mode === 'login' ? 'register' : 'login')"
          >
            {{ copy.switchLink }}
          </button>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch, type ComponentPublicInstance } from 'vue'
import { AlertTriangle, ArrowRight } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import AuthInput from './AuthInput.vue'
import {
  normalizeEmail,
  normalizeUsername,
  validateAuthForm,
  type AuthErrorView,
  type AuthField,
  type AuthFormValues,
  type AuthMode,
  type AuthSubmitPayload
} from './validation'

const props = withDefaults(
  defineProps<{ mode: AuthMode; loading?: boolean; serverError?: AuthErrorView | null }>(),
  { loading: false, serverError: null }
)

const emit = defineEmits<{
  'update:mode': [mode: AuthMode]
  submit: [payload: AuthSubmitPayload]
  'clear-server-error': []
}>()

const TABS = [
  { mode: 'login', label: 'Kirish' },
  { mode: 'register', label: "Ro'yxatdan o'tish" }
] as const satisfies readonly { mode: AuthMode; label: string }[]

const COPY = {
  login: {
    heading: 'Qaytganingizdan xursandmiz',
    sub: 'Laboratoriyani davom ettirish uchun kiring.',
    cta: 'Kirish',
    switchText: "Hisobingiz yo'qmi?",
    switchLink: "Ro'yxatdan o'ting"
  },
  register: {
    heading: 'Hisob yarating',
    sub: 'Bir daqiqada — va birinchi serveringiz tayyor.',
    cta: "Ro'yxatdan o'tish",
    switchText: 'Hisobingiz bormi?',
    switchLink: 'Kiring'
  }
} as const

const copy = computed(() => COPY[props.mode])

const values = reactive<AuthFormValues>({ username: '', email: '', password: '' })
const touched = reactive<Record<AuthField, boolean>>({ username: false, email: false, password: false })
const submitted = ref(false)

const usernameRef = ref<InstanceType<typeof AuthInput> | null>(null)
const emailRef = ref<InstanceType<typeof AuthInput> | null>(null)
const passwordRef = ref<InstanceType<typeof AuthInput> | null>(null)
const tabRefs = new Map<AuthMode, HTMLButtonElement>()

const clientErrors = computed(() => validateAuthForm(values, props.mode))

/** Client errors appear after the field was left once or a submit was attempted; server ones always. */
function fieldError(field: AuthField): string | undefined {
  const server = props.serverError?.fields?.[field]
  if (server) return server
  if (!submitted.value && !touched[field]) return undefined
  return clientErrors.value[field]
}

const formError = computed(() => props.serverError?.form)

function touch(field: AuthField) {
  // An empty field the user merely tabbed through is not an error yet.
  if (field === 'password' ? values.password : values[field].trim()) touched[field] = true
}

function onUsernameBlur() {
  values.username = normalizeUsername(values.username)
  touch('username')
}

function onEmailBlur() {
  values.email = normalizeEmail(values.email)
  touch('email')
}

// Any edit invalidates the previous server answer.
watch(values, () => {
  if (props.serverError) emit('clear-server-error')
})

function setMode(next: AuthMode) {
  if (next === props.mode) return
  submitted.value = false
  touched.username = false
  touched.email = false
  touched.password = false
  values.password = ''
  emit('clear-server-error')
  emit('update:mode', next)
}

function setTabRef(mode: AuthMode, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLButtonElement) tabRefs.set(mode, el)
}

function onTabKeydown(event: KeyboardEvent) {
  const order: AuthMode[] = ['login', 'register']
  const index = order.indexOf(props.mode)
  let next: AuthMode | undefined
  if (event.key === 'ArrowRight') next = order[(index + 1) % order.length]
  else if (event.key === 'ArrowLeft') next = order[(index - 1 + order.length) % order.length]
  else if (event.key === 'Home') next = order[0]
  else if (event.key === 'End') next = order[order.length - 1]
  if (!next) return
  event.preventDefault()
  setMode(next)
  void nextTick(() => tabRefs.get(next)?.focus())
}

function focusFirstInvalid() {
  const errors = clientErrors.value
  if (errors.username) usernameRef.value?.focus()
  else if (errors.email) emailRef.value?.focus()
  else if (errors.password) passwordRef.value?.focus()
}

function onSubmit() {
  if (props.loading) return
  values.username = normalizeUsername(values.username)
  values.email = normalizeEmail(values.email)
  submitted.value = true
  if (Object.keys(clientErrors.value).length > 0) {
    focusFirstInvalid()
    return
  }
  emit('submit', { mode: props.mode, ...values })
}

// Server field errors move focus to the offending input so screen readers hear the message.
watch(
  () => props.serverError,
  (err) => {
    const fields = err?.fields
    if (!fields) return
    void nextTick(() => {
      if (fields.username) usernameRef.value?.focus()
      else if (fields.email) emailRef.value?.focus()
      else if (fields.password) passwordRef.value?.focus()
    })
  }
)
</script>
