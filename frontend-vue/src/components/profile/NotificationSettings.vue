<!--
  NotificationSettings — "Bildirishnomalar" section. The backend has no notification settings yet,
  so every switch is visible but disabled and marked "tez orada".
  Props: none.
-->
<template>
  <section aria-labelledby="notif-title" class="rounded-panel border border-border bg-[rgba(15,21,32,0.88)] px-5 pb-2 pt-5 sm:px-6 sm:pt-6">
    <div class="mb-3 flex flex-wrap items-center gap-2.5">
      <h2 id="notif-title" class="font-mono text-lg font-semibold text-text">Bildirishnomalar</h2>
      <Badge color="#F5B342">tez orada</Badge>
    </div>
    <p class="mb-2 text-[13px] text-muted">Bildirishnoma sozlamalari hali ishga tushirilmagan. Ular tez orada shu yerda paydo bo'ladi.</p>
    <ul>
      <li
        v-for="item in ITEMS"
        :key="item.id"
        class="flex items-center justify-between gap-4 border-t border-divider py-3.5"
      >
        <span class="min-w-0">
          <span class="block font-medium text-text-2">{{ item.label }}</span>
          <span :id="`notif-${item.id}-desc`" class="block text-[13px] text-muted">{{ item.description }}</span>
        </span>
        <ToggleSwitch
          :model-value="false"
          :label="`${item.label} (tez orada)`"
          :describedby="`notif-${item.id}-desc`"
          disabled
        />
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import Badge from '@/components/ui/Badge.vue'
import ToggleSwitch from './ToggleSwitch.vue'

const ITEMS = [
  { id: 'checks', label: 'Tekshiruv natijalari', description: 'Har bir tekshiruvdan keyin qisqa xabar' },
  { id: 'expire', label: 'Sessiya tugashi', description: 'Tugashiga 5 daqiqa qolganda ogohlantirish' },
  { id: 'weekly', label: 'Haftalik xulosa', description: 'Emailga haftalik progress' }
] as const
</script>
