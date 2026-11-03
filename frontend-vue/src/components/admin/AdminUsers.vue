<!--
  AdminUsers — searchable user table with role / active / delete actions.
  Every action goes through ConfirmDialog, is applied optimistically and rolled back on error.
  The signed-in admin's own row has its actions disabled (the backend refuses them too).
  Data: GET /api/admin/users, PUT /api/admin/users/:id {role?|active?}, DELETE /api/admin/users/:id.
  Emits: count(n) (for the side nav), changed (after a successful mutation)
  Exposes: refresh()
-->
<template>
  <section class="overflow-hidden rounded-panel border border-border bg-[rgba(15,21,32,0.88)]" aria-label="Foydalanuvchilar ro'yxati">
    <div class="flex flex-wrap items-center gap-3 border-b border-divider px-4 py-3">
      <label class="relative block min-w-0 flex-[1_1_240px] sm:max-w-[340px]">
        <span class="sr-only">Foydalanuvchi qidirish</span>
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-faint" aria-hidden="true" />
        <input
          v-model="query"
          type="search"
          class="input !h-10 pl-9 font-mono text-[13px]"
          placeholder="grep user / email"
          autocomplete="off"
          spellcheck="false"
        />
      </label>
      <span class="ml-auto font-mono text-meta text-muted" aria-live="polite">
        <template v-if="loaded">{{ filtered.length }} / {{ users.length }} ta</template>
      </span>
    </div>

    <div v-if="error && !loaded" class="p-4">
      <SectionError :message="error" command="GET /api/admin/users" :retrying="loading" @retry="refresh" />
    </div>

    <div v-else-if="loaded && filtered.length === 0" class="p-4">
      <EmptyState
        :icon="UserX"
        :command="query ? `grep '${query}' users → 0` : 'users: 0'"
        :title="query ? 'Hech narsa topilmadi' : 'Foydalanuvchilar yo\'q'"
        :description="query ? 'Boshqa ism yoki email bilan qidirib ko\'ring.' : undefined"
      >
        <template v-if="query" #actions>
          <Button size="sm" @click="query = ''">Qidiruvni tozalash</Button>
        </template>
      </EmptyState>
    </div>

    <div v-else class="relative overflow-x-auto">
      <table class="w-full min-w-[920px] border-collapse">
        <caption class="sr-only">Foydalanuvchilar: rol, holat, XP, yakunlangan vazifalar va oxirgi faollik</caption>
        <thead>
          <tr class="th text-left">
            <th scope="col" class="px-5 py-3 font-medium">Foydalanuvchi</th>
            <th scope="col" class="px-4 py-3 font-medium">Rol</th>
            <th scope="col" class="px-4 py-3 font-medium">Holat</th>
            <th scope="col" class="px-4 py-3 text-right font-medium">XP</th>
            <th scope="col" class="px-4 py-3 text-right font-medium">Yakunlagan</th>
            <th scope="col" class="px-4 py-3 font-medium">Oxirgi faollik</th>
            <th scope="col" class="py-3 pl-4 pr-5 text-right font-medium">Amallar</th>
          </tr>
        </thead>
        <tbody v-if="!loaded">
          <tr v-for="i in 6" :key="i" class="border-t border-divider">
            <td class="px-5 py-3"><Skeleton width="140px" /><Skeleton width="180px" height="11px" class="mt-1.5" /></td>
            <td class="px-4 py-3"><Skeleton width="70px" height="20px" rounded="rounded-full" /></td>
            <td class="px-4 py-3"><Skeleton width="60px" height="20px" rounded="rounded-full" /></td>
            <td class="px-4 py-3"><Skeleton width="40px" class="ml-auto" /></td>
            <td class="px-4 py-3"><Skeleton width="30px" class="ml-auto" /></td>
            <td class="px-4 py-3"><Skeleton width="80px" /></td>
            <td class="py-3 pl-4 pr-5"><Skeleton width="220px" height="32px" class="ml-auto" /></td>
          </tr>
        </tbody>
        <tbody v-else>
          <tr v-for="u in filtered" :key="u.id" class="border-t border-divider" :aria-busy="busy.has(u.id) || undefined">
            <th scope="row" class="px-5 py-3 text-left font-normal">
              <span class="block font-mono font-medium text-text">
                {{ u.username }}
                <span v-if="u.id === selfId" class="ml-1 font-sans text-xs font-normal text-muted">(siz)</span>
              </span>
              <span class="block max-w-[260px] truncate text-meta text-muted" :title="u.email">{{ u.email || '—' }}</span>
            </th>
            <td class="px-4 py-3">
              <Badge v-if="u.role === 'admin'" color="#F5B342">Admin</Badge>
              <Badge v-else variant="open" :icon="false" label="O'quvchi" />
            </td>
            <td class="px-4 py-3">
              <Badge :color="u.active ? '#3DDC97' : '#F87171'">{{ u.active ? 'Faol' : 'Bloklangan' }}</Badge>
            </td>
            <td class="px-4 py-3 text-right font-mono text-[13px] text-accent">{{ count(u.score) }}</td>
            <td class="px-4 py-3 text-right font-mono text-[13px] text-text-2">{{ u.completed }}</td>
            <td class="px-4 py-3 text-muted">
              <time v-if="u.last_active" :datetime="u.last_active" :title="fullDateTime(u.last_active)">{{ relativeTime(u.last_active, now) }}</time>
              <span v-else>—</span>
            </td>
            <td class="whitespace-nowrap py-3 pl-4 pr-5 text-right">
              <div class="inline-flex items-center gap-1.5">
                <button
                  type="button"
                  :class="ACTION_BTN"
                  :disabled="isLocked(u)"
                  :title="u.id === selfId ? 'O\'z rolingizni o\'zgartira olmaysiz' : undefined"
                  @click="ask('role', u)"
                >
                  {{ u.role === 'admin' ? "O'quvchi qilish" : 'Admin qilish' }}
                </button>
                <button
                  type="button"
                  :class="ACTION_BTN"
                  :disabled="isLocked(u)"
                  :title="u.id === selfId ? 'O\'zingizni bloklay olmaysiz' : undefined"
                  @click="ask('active', u)"
                >
                  {{ u.active ? 'Bloklash' : 'Faollashtirish' }}
                </button>
                <button
                  type="button"
                  :class="[ACTION_BTN, 'w-8 !px-0 !text-danger hover:!border-danger/60']"
                  :disabled="isLocked(u)"
                  :aria-label="`${u.username} ni o'chirish`"
                  :title="u.id === selfId ? 'O\'zingizni o\'chira olmaysiz' : 'O\'chirish'"
                  @click="ask('delete', u)"
                >
                  <Trash2 class="h-4 w-4" aria-hidden="true" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmDialog
      :open="pending !== null"
      :title="dialog.title"
      :message="dialog.message"
      :confirm-label="dialog.confirm"
      :variant="dialog.variant"
      @update:open="(v) => { if (!v) pending = null }"
      @confirm="confirm"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Search, Trash2, UserX } from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import SectionError from './SectionError.vue'
import { count, fullDateTime, relativeTime } from './format'
import { api, errorMessage } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import type { AdminUser, AdminUserUpdate } from '@/types'

type ActionKind = 'role' | 'active' | 'delete'

const emit = defineEmits<{ count: [value: number]; changed: [] }>()

const ACTION_BTN =
  'inline-flex h-8 items-center justify-center rounded-[7px] border border-border-strong bg-surface-solid px-2.5 text-[13px] text-text-2 transition-colors hover:border-border-stronger hover:text-text disabled:cursor-not-allowed disabled:opacity-40'

const auth = useAuthStore()
const toast = useToast()

const users = ref<AdminUser[]>([])
const loaded = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)
const query = ref('')
const busy = ref(new Set<number>())
const pending = ref<{ kind: ActionKind; user: AdminUser } | null>(null)
const now = ref(Date.now())
let clock: ReturnType<typeof setInterval> | null = null

const selfId = computed(() => auth.user?.id ?? null)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return users.value
  return users.value.filter((u) => u.username.toLowerCase().includes(q) || (u.email ?? '').toLowerCase().includes(q))
})

function isLocked(u: AdminUser): boolean {
  return u.id === selfId.value || busy.value.has(u.id)
}

async function refresh(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    users.value = await api.adminUsers()
    loaded.value = true
    emit('count', users.value.length)
  } catch (err) {
    error.value = errorMessage(err, "Foydalanuvchilarni yuklab bo'lmadi.")
    if (loaded.value) toast.error(error.value)
  } finally {
    loading.value = false
    now.value = Date.now()
  }
}

function ask(kind: ActionKind, user: AdminUser) {
  if (isLocked(user)) return
  pending.value = { kind, user }
}

const dialog = computed(() => {
  const p = pending.value
  if (!p) return { title: '', message: '', confirm: 'Tasdiqlash', variant: 'danger' as const }
  const name = p.user.username
  if (p.kind === 'role') {
    return p.user.role === 'admin'
      ? { title: 'Admin huquqini olish', message: `${name} oddiy o'quvchiga aylantiriladi va admin paneliga kira olmaydi.`, confirm: "O'quvchi qilish", variant: 'danger' as const }
      : { title: 'Admin huquqini berish', message: `${name} admin paneliga to'liq kirish huquqini oladi.`, confirm: 'Admin qilish', variant: 'primary' as const }
  }
  if (p.kind === 'active') {
    return p.user.active
      ? { title: 'Foydalanuvchini bloklash', message: `${name} tizimga kira olmaydi va faol sessiyalari ishlamay qoladi.`, confirm: 'Bloklash', variant: 'danger' as const }
      : { title: 'Foydalanuvchini faollashtirish', message: `${name} yana tizimga kira oladi.`, confirm: 'Faollashtirish', variant: 'primary' as const }
  }
  return {
    title: "Foydalanuvchini o'chirish",
    message: `${name} va uning barcha natijalari o'chiriladi. Bu amalni qaytarib bo'lmaydi.`,
    confirm: "O'chirish",
    variant: 'danger' as const
  }
})

function replaceUser(id: number, next: AdminUser) {
  const idx = users.value.findIndex((u) => u.id === id)
  if (idx >= 0) users.value.splice(idx, 1, next)
}

async function confirm() {
  const p = pending.value
  if (!p) return
  pending.value = null
  const { kind, user } = p
  const index = users.value.findIndex((u) => u.id === user.id)
  if (index < 0) return
  const before: AdminUser = { ...users.value[index] }
  busy.value.add(user.id)

  try {
    if (kind === 'delete') {
      users.value.splice(index, 1)
      emit('count', users.value.length)
      await api.adminDeleteUser(user.id)
      toast.success(`${user.username} o'chirildi`)
    } else {
      const patch: AdminUserUpdate = kind === 'role' ? { role: before.role === 'admin' ? 'student' : 'admin' } : { active: !before.active }
      replaceUser(user.id, { ...before, ...patch })
      const updated = await api.adminUpdateUser(user.id, patch)
      const current = users.value.find((u) => u.id === user.id)
      if (current) replaceUser(user.id, { ...current, role: updated.role, active: updated.active })
      toast.success(
        kind === 'role'
          ? `${user.username}: rol "${updated.role === 'admin' ? 'Admin' : "O'quvchi"}" ga o'zgardi`
          : `${user.username} ${updated.active ? 'faollashtirildi' : 'bloklandi'}`
      )
    }
    emit('changed')
  } catch (err) {
    // Roll back the optimistic change.
    if (kind === 'delete') {
      if (!users.value.some((u) => u.id === before.id)) users.value.splice(Math.min(index, users.value.length), 0, before)
      emit('count', users.value.length)
    } else {
      replaceUser(user.id, before)
    }
    toast.error(errorMessage(err, "Amalni bajarib bo'lmadi."))
  } finally {
    busy.value.delete(user.id)
  }
}

onMounted(() => {
  void refresh()
  clock = setInterval(() => (now.value = Date.now()), 60_000)
})
onBeforeUnmount(() => {
  if (clock) clearInterval(clock)
})

defineExpose({ refresh, loading })
</script>
