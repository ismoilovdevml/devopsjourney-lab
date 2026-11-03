<!--
  ProfileForm — "Profil" section: username (read-only), email, GitHub username, goal (<= 200 chars).
  Client validation mirrors the backend (see ./profile.ts). Save/Cancel appear enabled only when the
  form is dirty. On success: PUT /api/me -> auth.setUser + success toast.
  Props:
    user: User
  Emits: dirty(boolean)  (so the page can warn before leaving)
-->
<template>
  <section aria-labelledby="profile-form-title" class="rounded-panel border border-border bg-[rgba(15,21,32,0.88)] p-5 sm:p-6">
    <form class="flex flex-col gap-5" novalidate @submit.prevent="save">
      <div>
        <h2 id="profile-form-title" class="font-mono text-lg font-semibold text-text">Profil</h2>
        <p class="mt-0.5 text-[13px] text-muted">Bu ma'lumotlar faqat sizga va administratorga ko'rinadi.</p>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label for="pf-username" class="label">Foydalanuvchi nomi</label>
          <input
            id="pf-username"
            :value="user.username"
            type="text"
            class="input h-[46px] cursor-not-allowed font-mono text-muted"
            readonly
            aria-describedby="pf-username-hint"
          />
          <p id="pf-username-hint" class="mt-1.5 text-xs text-muted">Foydalanuvchi nomini o'zgartirib bo'lmaydi.</p>
        </div>

        <div>
          <label for="pf-email" class="label">Email</label>
          <input
            id="pf-email"
            v-model="form.email"
            type="email"
            autocomplete="email"
            :maxlength="EMAIL_MAX"
            class="input h-[46px] font-mono"
            :aria-invalid="!!shownErrors.email"
            :aria-describedby="shownErrors.email ? 'pf-email-err' : undefined"
            required
            @blur="touched.email = true"
          />
          <p v-if="shownErrors.email" id="pf-email-err" class="mt-1.5 text-xs text-danger">{{ shownErrors.email }}</p>
        </div>

        <div class="md:col-span-2">
          <label for="pf-github" class="label">GitHub</label>
          <div class="relative">
            <span class="pointer-events-none absolute inset-y-0 left-3.5 flex items-center font-mono text-sm text-faint" aria-hidden="true">github.com/</span>
            <input
              id="pf-github"
              v-model="form.github"
              type="text"
              autocomplete="off"
              autocapitalize="off"
              spellcheck="false"
              placeholder="username"
              class="input h-[46px] pl-[106px] font-mono"
              :aria-invalid="!!shownErrors.github"
              :aria-describedby="shownErrors.github ? 'pf-github-err' : 'pf-github-hint'"
              @blur="touched.github = true"
            />
          </div>
          <p v-if="shownErrors.github" id="pf-github-err" class="mt-1.5 text-xs text-danger">{{ shownErrors.github }}</p>
          <p v-else id="pf-github-hint" class="mt-1.5 text-xs text-muted">Ixtiyoriy. Faqat foydalanuvchi nomi yoki profil havolasi.</p>
        </div>

        <div class="md:col-span-2">
          <div class="flex items-baseline justify-between gap-3">
            <label for="pf-goal" class="label">Maqsad</label>
            <span
              id="pf-goal-count"
              :class="['font-mono text-xs', goalLength > GOAL_MAX ? 'text-danger' : 'text-muted']"
              aria-live="polite"
            >
              {{ goalLength }}/{{ GOAL_MAX }}
            </span>
          </div>
          <textarea
            id="pf-goal"
            v-model="form.goal"
            rows="3"
            placeholder="Masalan: 6 oyda Junior DevOps Engineer bo'lish"
            class="input h-auto min-h-[92px] resize-y py-2.5 leading-relaxed"
            :aria-invalid="!!shownErrors.goal"
            :aria-describedby="shownErrors.goal ? 'pf-goal-err pf-goal-count' : 'pf-goal-count'"
            @blur="touched.goal = true"
          />
          <p v-if="shownErrors.goal" id="pf-goal-err" class="mt-1.5 text-xs text-danger">{{ shownErrors.goal }}</p>
        </div>
      </div>

      <p v-if="formError" role="alert" class="rounded-btn border border-danger/35 bg-danger/[0.06] px-3.5 py-2.5 text-[13px] text-danger">
        {{ formError }}
      </p>

      <div class="flex flex-wrap items-center justify-end gap-2.5">
        <span v-if="dirty" class="mr-auto text-[13px] text-warn">Saqlanmagan o'zgarishlar bor</span>
        <Button variant="ghost" :disabled="!dirty || saving" @click="reset">Bekor qilish</Button>
        <Button type="submit" variant="primary" :disabled="!dirty" :loading="saving">Saqlash</Button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import { api, errorMessage, isApiStatus, ApiError } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import type { UpdateMeRequest, User } from '@/types'
import {
  EMAIL_MAX,
  GOAL_MAX,
  normalizeProfile,
  profileServerError,
  runeLength,
  validateProfile,
  type ProfileErrors,
  type ProfileFields
} from './profile'

const props = defineProps<{ user: User }>()
const emit = defineEmits<{ dirty: [value: boolean] }>()

const auth = useAuthStore()
const toast = useToast()

const fromUser = (u: User): ProfileFields => ({ email: u.email ?? '', github: u.github ?? '', goal: u.goal ?? '' })

const form = reactive<ProfileFields>(fromUser(props.user))
const touched = reactive<Record<keyof ProfileFields, boolean>>({ email: false, github: false, goal: false })
const submitted = ref(false)
const saving = ref(false)
const serverErrors = ref<ProfileErrors>({})
const formError = ref('')

const baseline = computed(() => normalizeProfile(fromUser(props.user)))
const current = computed(() => normalizeProfile(form))
const changed = computed(() => {
  const body: UpdateMeRequest = {}
  if (current.value.email !== baseline.value.email) body.email = current.value.email
  if (current.value.github !== baseline.value.github) body.github = current.value.github
  if (current.value.goal !== baseline.value.goal) body.goal = current.value.goal
  return body
})
const dirty = computed(() => Object.keys(changed.value).length > 0)
const goalLength = computed(() => runeLength(form.goal.trim()))

const clientErrors = computed(() => validateProfile(form))
const shownErrors = computed<ProfileErrors>(() => {
  const out: ProfileErrors = {}
  for (const key of ['email', 'github', 'goal'] as const) {
    const msg = serverErrors.value[key] ?? clientErrors.value[key]
    // The goal counter is live, so its error shows as soon as the limit is crossed.
    if (msg && (submitted.value || touched[key] || key === 'goal' || serverErrors.value[key])) out[key] = msg
  }
  return out
})

// Any edit clears stale server feedback.
watch(
  () => ({ ...form }),
  () => {
    serverErrors.value = {}
    formError.value = ''
  }
)
watch(dirty, (v) => emit('dirty', v), { immediate: true })

// If the user object is replaced (e.g. after save), re-sync a clean form.
watch(
  () => props.user,
  (u) => {
    if (!saving.value) Object.assign(form, fromUser(u))
  }
)

function reset() {
  Object.assign(form, fromUser(props.user))
  touched.email = touched.github = touched.goal = false
  submitted.value = false
  serverErrors.value = {}
  formError.value = ''
}

async function save() {
  submitted.value = true
  if (!dirty.value || Object.keys(clientErrors.value).length > 0) {
    const first = (['email', 'github', 'goal'] as const).find((k) => clientErrors.value[k])
    if (first) document.getElementById(`pf-${first}`)?.focus()
    return
  }
  saving.value = true
  formError.value = ''
  try {
    const updated = await api.updateMe(changed.value)
    auth.setUser(updated)
    Object.assign(form, fromUser(updated))
    submitted.value = false
    touched.email = touched.github = touched.goal = false
    toast.success('Profil saqlandi')
  } catch (err) {
    if (isApiStatus(err, 409)) {
      serverErrors.value = { email: 'Bu email boshqa hisobda ishlatilmoqda.' }
    } else if (isApiStatus(err, 400)) {
      const msg = err instanceof ApiError ? err.body?.message ?? err.message : ''
      const mapped = profileServerError(msg)
      if (mapped) serverErrors.value = mapped
      else formError.value = errorMessage(err, "Ma'lumotlar noto'g'ri. Tekshirib qayta urinib ko'ring.")
    } else {
      formError.value = errorMessage(err, "Profilni saqlab bo'lmadi. Keyinroq urinib ko'ring.")
    }
  } finally {
    saving.value = false
  }
}
</script>
