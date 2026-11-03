<!--
  AdminView — STUB. Owned by the Admin page agent; see design/Admin.dc.html.
  Own chrome (/admin header, sudo badge, side nav). Reference implementation: views/_legacy/AdminView.vue.
  API: api.adminStats/adminUsers/adminUpdateUser/adminDeleteUser/adminSessions/adminTerminateSession/
       adminActivity/adminResources/adminReloadTasks.
-->
<template>
  <div class="flex min-h-screen flex-col">
    <header class="flex items-center gap-4 border-b border-border bg-[rgba(10,14,20,0.82)] px-6 py-3 backdrop-blur-md">
      <AppLogo suffix="/admin" />
      <Badge color="#F5B342">sudo</Badge>
    </header>
    <main id="main" class="container-page flex flex-1 flex-col gap-6 py-8">
      <PageHeader command="$ sudo systemctl status lab" title="Boshqaruv paneli" />
      <EmptyState command="TODO" title="Admin paneli ishlab chiqilmoqda" description="Bu sahifa tez orada tayyor bo'ladi." />
    </main>
  </div>
</template>

<script setup lang="ts">
import AppLogo from '@/components/ui/AppLogo.vue'
import Badge from '@/components/ui/Badge.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
</script>
