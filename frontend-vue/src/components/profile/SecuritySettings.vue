<!--
  SecuritySettings — "Xavfsizlik" section:
    - password change (current / new / confirm) with a strength meter; client rules mirror
      backend ValidatePasswordStrength. POST /api/me/password: 204 -> toast + clear,
      401 -> "wrong current password" on that field, 400 -> weak-password message.
    - sessions: "log out everywhere" (disabled, tez orada) and "Hisobdan chiqish" (auth.logout).
    - danger zone: reset own progress (DELETE /api/progress) behind a ConfirmDialog.
  Props:
    username: string
  Emits: progress-reset (after a successful reset, so the page can reload progress)
-->
<template>
  <div class="flex flex-col gap-6">
    <section aria-labelledby="pw-title" class="rounded-panel border border-border bg-[rgba(15,21,32,0.88)] p-5 sm:p-6">
      <form class="flex flex-col gap-4" novalidate @submit.prevent="submit">
        <div>
          <h2 id="pw-title" class="font-mono text-lg font-semibold text-text">Xavfsizlik</h2>
          <p class="mt-0.5 text-[13px] text-muted">Parolni yangilash. Kamida {{ PASSWORD_MIN }} belgi, foydalanuvchi nomingizdan farqli.</p>
        </div>

        <!-- Hidden username helps password managers associate the new password with the account. -->
        <input type="text" name="username" autocomplete="username" :value="username" class="hidden" readonly tabindex="-1" aria-hidden="true" />

        <div class="max-w-[420px]">
          <label for="pw-current" class="label">Joriy parol</label>
          <input
            id="pw-current"
            v-model="current"
            :type="reveal ? 'text' : 'password'"
            autocomplete="current-password"
            class="input h-[46px]"
            :aria-invalid="!!errors.current"
            :aria-describedby="errors.current ? 'pw-current-err' : undefined"
            @input="serverError.current = ''"
          />
          <p v-if="errors.current" id="pw-current-err" class="mt-1.5 text-xs text-danger">{{ errors.current }}</p>
        </div>

        <div class="max-w-[420px]">
          <label for="pw-new" class="label">Yangi parol</label>
          <input
            id="pw-new"
            v-model="next"
            :type="reveal ? 'text' : 'password'"
            autocomplete="new-password"
            class="input h-[46px]"
            :aria-invalid="!!errors.next"
            :aria-describedby="errors.next ? 'pw-new-err pw-strength' : 'pw-strength'"
            @input="serverError.next = ''"
            @blur="touched.next = true"
          />
          <div class="mt-2 flex items-center gap-3">
            <div class="flex flex-1 gap-1" aria-hidden="true">
              <span
                v-for="i in 4"
                :key="i"
                class="h-1.5 flex-1 rounded-full transition-colors"
                :style="{ background: i <= strength.score ? strength.color : '#1B2535' }"
              />
            </div>
            <span id="pw-strength" class="w-[72px] text-right text-xs font-medium" :style="{ color: strength.score ? strength.color : '#94A3B8' }">
              <span class="sr-only">Parol kuchi: </span>{{ strength.label || '—' }}
            </span>
          </div>
          <p v-if="errors.next" id="pw-new-err" class="mt-1.5 text-xs text-danger">{{ errors.next }}</p>
        </div>

        <div class="max-w-[420px]">
          <label for="pw-confirm" class="label">Yangi parolni tasdiqlang</label>
          <input
            id="pw-confirm"
            v-model="confirm"
            :type="reveal ? 'text' : 'password'"
            autocomplete="new-password"
            class="input h-[46px]"
            :aria-invalid="!!errors.confirm"
            :aria-describedby="errors.confirm ? 'pw-confirm-err' : undefined"
            @blur="touched.confirm = true"
          />
          <p v-if="errors.confirm" id="pw-confirm-err" class="mt-1.5 text-xs text-danger">{{ errors.confirm }}</p>
        </div>

        <label class="flex w-max cursor-pointer items-center gap-2 text-[13px] text-text-2">
          <input v-model="reveal" type="checkbox" class="h-4 w-4 accent-[#3DDC97]" />
          Parollarni ko'rsatish
        </label>

        <p v-if="formError" role="alert" class="max-w-[420px] rounded-btn border border-danger/35 bg-danger/[0.06] px-3.5 py-2.5 text-[13px] text-danger">
          {{ formError }}
        </p>

        <div>
          <Button type="submit" variant="primary" :loading="saving" :disabled="!current && !next && !confirm">Parolni yangilash</Button>
        </div>
      </form>
    </section>

    <section aria-labelledby="sessions-title" class="rounded-panel border border-border bg-[rgba(15,21,32,0.88)] p-5 sm:p-6">
      <h2 id="sessions-title" class="font-mono text-base font-semibold text-text">Kirish seanslari</h2>
      <div class="mt-3 flex flex-col divide-y divide-divider">
        <div class="flex flex-wrap items-center justify-between gap-3 py-3">
          <span class="min-w-0">
            <span class="flex items-center gap-2 font-medium text-text-2">
              Barcha qurilmalardan chiqish
              <Badge color="#F5B342">tez orada</Badge>
            </span>
            <span class="block text-[13px] text-muted">Boshqa brauzer va qurilmalardagi seanslarni yakunlash.</span>
          </span>
          <Button size="sm" disabled>Hammasidan chiqish</Button>
        </div>
        <div class="flex flex-wrap items-center justify-between gap-3 py-3">
          <span class="min-w-0">
            <span class="block font-medium text-text-2">Shu qurilmadan chiqish</span>
            <span class="block text-[13px] text-muted">Joriy brauzerdagi seans yakunlanadi.</span>
          </span>
          <Button size="sm" :icon="LogOut" :loading="loggingOut" @click="logout">Hisobdan chiqish</Button>
        </div>
      </div>
    </section>

    <section
      aria-labelledby="danger-title"
      class="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-danger/35 bg-danger/[0.06] p-4"
    >
      <span>
        <span id="danger-title" class="block font-semibold text-[#FCA5A5]">Progressni nolga qaytarish</span>
        <span class="block text-[13px] text-text-2">Barcha ball va urinishlar o'chiriladi</span>
      </span>
      <Button variant="danger" size="sm" class="h-10" @click="resetOpen = true">Qaytarish</Button>
    </section>

    <ConfirmDialog
      v-model:open="resetOpen"
      title="Progressni nolga qaytarasizmi?"
      message="Barcha ball, urinishlar va yakunlangan vazifalar o'chiriladi. Bu amalni ortga qaytarib bo'lmaydi."
      confirm-label="Ha, qaytarish"
      :loading="resetting"
      @confirm="resetProgress"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { LogOut } from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import { api, ApiError, errorMessage, isApiStatus } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import {
  PASSWORD_MIN,
  passwordRuleError,
  passwordServerError,
  passwordStrength,
  type PasswordStrength
} from './profile'

const props = defineProps<{ username: string }>()
const emit = defineEmits<{ 'progress-reset': [] }>()

const auth = useAuthStore()
const router = useRouter()
const toast = useToast()

const current = ref('')
const next = ref('')
const confirm = ref('')
const reveal = ref(false)
const saving = ref(false)
const submitted = ref(false)
const touched = reactive({ next: false, confirm: false })
const serverError = reactive({ current: '', next: '' })
const formError = ref('')

// A server-side rejection always reads as weak, even if the local heuristic scored higher.
const strength = computed<PasswordStrength>(() =>
  serverError.next ? { score: 1, label: 'Zaif', color: '#F87171' } : passwordStrength(next.value, props.username)
)

const clientErrors = computed(() => ({
  current: current.value ? '' : 'Joriy parolni kiriting.',
  next: next.value ? passwordRuleError(next.value, props.username, current.value) ?? '' : 'Yangi parolni kiriting.',
  confirm: !confirm.value
    ? 'Yangi parolni qayta kiriting.'
    : confirm.value !== next.value
      ? 'Parollar mos kelmadi.'
      : ''
}))

const errors = computed(() => ({
  current: serverError.current || (submitted.value ? clientErrors.value.current : ''),
  next: serverError.next || (submitted.value || (touched.next && next.value) ? clientErrors.value.next : ''),
  confirm: submitted.value || (touched.confirm && confirm.value) ? clientErrors.value.confirm : ''
}))

function clearForm() {
  current.value = next.value = confirm.value = ''
  submitted.value = false
  touched.next = touched.confirm = false
  serverError.current = serverError.next = ''
  formError.value = ''
  reveal.value = false
}

async function submit() {
  submitted.value = true
  formError.value = ''
  const ce = clientErrors.value
  const firstBad = ce.current ? 'pw-current' : ce.next ? 'pw-new' : ce.confirm ? 'pw-confirm' : null
  if (firstBad) {
    document.getElementById(firstBad)?.focus()
    return
  }
  saving.value = true
  try {
    await api.changePassword({ current_password: current.value, new_password: next.value })
    clearForm()
    toast.success('Parol yangilandi')
  } catch (err) {
    if (isApiStatus(err, 401)) {
      serverError.current = "Joriy parol noto'g'ri."
      document.getElementById('pw-current')?.focus()
    } else if (isApiStatus(err, 400)) {
      const msg = err instanceof ApiError ? err.body?.message ?? '' : ''
      serverError.next = msg ? passwordServerError(msg) : "Parol talablarga javob bermaydi."
      document.getElementById('pw-new')?.focus()
    } else {
      formError.value = errorMessage(err, "Parolni yangilab bo'lmadi. Keyinroq urinib ko'ring.")
    }
  } finally {
    saving.value = false
  }
}

const loggingOut = ref(false)
async function logout() {
  loggingOut.value = true
  try {
    await auth.logout()
    await router.replace({ name: 'login' })
  } finally {
    loggingOut.value = false
  }
}

const resetOpen = ref(false)
const resetting = ref(false)
async function resetProgress() {
  resetting.value = true
  try {
    await api.resetProgress()
    resetOpen.value = false
    toast.success('Progress nolga qaytarildi')
    emit('progress-reset')
  } catch (err) {
    toast.error(errorMessage(err, "Progressni qaytarib bo'lmadi."))
  } finally {
    resetting.value = false
  }
}
</script>
