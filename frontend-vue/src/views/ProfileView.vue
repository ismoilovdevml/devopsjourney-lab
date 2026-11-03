<!--
  ProfileView — /profile?tab=profile|appearance|notifications|security
  Header card (user + level from GET /api/progress) and a settings side nav that becomes horizontal
  scrollable tabs on narrow screens. Sections stay mounted (v-show) so unsaved form edits survive
  switching tabs; a beforeunload guard warns while the profile form is dirty.
-->
<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-[1100px] flex-col gap-6">
      <ProfileHeaderCard
        v-if="auth.user"
        :user="auth.user"
        :progress="progress"
        :loading="progressLoading"
        :error="progressError"
        @retry="loadProgress"
      />

      <div class="flex flex-col gap-6 md:flex-row md:items-start">
        <nav
          ref="navEl"
          aria-label="Sozlamalar bo'limlari"
          class="-mx-4 flex gap-1 overflow-x-auto border-y border-border bg-surface px-4 py-2 sm:mx-0 sm:rounded-card sm:border md:sticky md:top-[84px] md:w-[240px] md:shrink-0 md:flex-col md:overflow-visible md:p-2"
        >
          <RouterLink
            v-for="s in PROFILE_SECTIONS"
            :key="s.id"
            :to="{ query: { ...route.query, tab: s.id } }"
            replace
            :aria-current="s.id === active ? 'page' : undefined"
            :class="[
              'flex h-10 shrink-0 items-center gap-2.5 whitespace-nowrap rounded-lg px-3 font-medium transition-colors',
              s.id === active
                ? 'bg-raised text-text shadow-[inset_0_-2px_0_#3DDC97] md:shadow-[inset_2px_0_0_#3DDC97]'
                : 'text-muted hover:bg-surface-3 hover:text-text'
            ]"
          >
            <component :is="ICONS[s.id]" class="h-4 w-4 shrink-0" aria-hidden="true" />
            {{ s.label }}
            <span v-if="s.id === 'profile' && profileDirty" class="h-1.5 w-1.5 rounded-full bg-warn" aria-label="saqlanmagan" />
          </RouterLink>
          <button
            type="button"
            class="flex h-10 shrink-0 items-center gap-2.5 whitespace-nowrap rounded-lg px-3 font-medium text-danger transition-colors hover:bg-danger/10 md:mt-2 md:rounded-t-none md:border-t md:border-border"
            :disabled="loggingOut"
            @click="logout"
          >
            <LogOut class="h-4 w-4 shrink-0" aria-hidden="true" />
            Chiqish
          </button>
        </nav>

        <div v-if="auth.user" class="min-w-0 flex-1">
          <div v-show="active === 'profile'">
            <ProfileForm :user="auth.user" @dirty="profileDirty = $event" />
          </div>
          <div v-show="active === 'appearance'">
            <AppearanceSettings />
          </div>
          <div v-show="active === 'notifications'">
            <NotificationSettings />
          </div>
          <div v-show="active === 'security'">
            <SecuritySettings :username="auth.user.username" @progress-reset="loadProgress" />
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bell, LogOut, Palette, ShieldCheck, UserRound } from 'lucide-vue-next'
import AppLayout from '@/components/ui/AppLayout.vue'
import ProfileHeaderCard from '@/components/profile/ProfileHeaderCard.vue'
import ProfileForm from '@/components/profile/ProfileForm.vue'
import AppearanceSettings from '@/components/profile/AppearanceSettings.vue'
import NotificationSettings from '@/components/profile/NotificationSettings.vue'
import SecuritySettings from '@/components/profile/SecuritySettings.vue'
import { PROFILE_SECTIONS, isProfileSection, type ProfileSection } from '@/components/profile/profile'
import { api } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import type { ProgressResponse } from '@/types'

const ICONS: Record<ProfileSection, Component> = {
  profile: UserRound,
  appearance: Palette,
  notifications: Bell,
  security: ShieldCheck
}

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const active = computed<ProfileSection>(() => (isProfileSection(route.query.tab) ? route.query.tab : 'profile'))

const progress = ref<ProgressResponse | null>(null)
const progressLoading = ref(true)
const progressError = ref(false)

async function loadProgress() {
  progressLoading.value = true
  progressError.value = false
  try {
    progress.value = await api.getProgress()
  } catch {
    progressError.value = true
  } finally {
    progressLoading.value = false
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

// Warn before closing/reloading the tab with unsaved profile edits.
const profileDirty = ref(false)
function onBeforeUnload(e: BeforeUnloadEvent) {
  if (!profileDirty.value) return
  e.preventDefault()
  e.returnValue = ''
}

// On narrow screens the nav is a horizontal scroller: keep the active tab visible.
const navEl = ref<HTMLElement | null>(null)
function revealActiveTab() {
  navEl.value?.querySelector<HTMLElement>('[aria-current="page"]')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}
watch(active, () => nextTick(revealActiveTab))

onMounted(() => {
  loadProgress()
  revealActiveTab()
  window.addEventListener('beforeunload', onBeforeUnload)
})
onBeforeUnmount(() => window.removeEventListener('beforeunload', onBeforeUnload))
</script>
