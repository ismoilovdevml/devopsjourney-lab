<!-- LeaderboardView — STUB. Owned by the page agent; see design/Leaderboard.dc.html and design/DESIGN.md. -->
<template>
  <AppLayout>
    <PageHeader command="$ sort -rn scores | head" title="Reyting" />
    <EmptyState command="TODO" title="Sahifa ishlab chiqilmoqda" description="Bu sahifa tez orada tayyor bo'ladi." />
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/ui/AppLayout.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
</script>
