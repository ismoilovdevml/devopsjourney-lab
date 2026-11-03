<!--
  AppFooterStatus — mono status bar: "● cluster: online · containers n/20 · image · version · date".
  Cluster state and version come from GET /healthz (fetched once per page load and shared).
  Props:
    containers?: { running: number; limit: number }   (shown only when provided, e.g. admin data)
    image?: string                                      (e.g. task image "ubuntu:22.04"; hidden when absent)
-->
<template>
  <footer class="border-t border-border bg-[rgba(10,14,20,0.9)] font-mono text-xs text-muted">
    <div class="container-page flex flex-wrap gap-x-5 gap-y-1 py-2">
      <span>
        <span :class="STATE_CLASS[health.state]" aria-hidden="true">●</span>
        cluster: {{ STATE_LABEL[health.state] }}
      </span>
      <span v-if="containers">containers {{ containers.running }}/{{ containers.limit }}</span>
      <span v-if="image">{{ image }}</span>
      <span class="ml-auto">{{ health.version ? `v${health.version.replace(/^v/, '')} · ` : '' }}{{ today }}</span>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { api } from '@/services/api'

defineProps<{
  containers?: { running: number; limit: number }
  image?: string
}>()

type ClusterState = 'checking' | 'online' | 'offline'

// Module-level cache: one health request per page load, shared by every footer instance.
const health = reactive<{ state: ClusterState; version: string }>({ state: 'checking', version: '' })
let requested = false

const STATE_LABEL: Record<ClusterState, string> = {
  checking: 'tekshirilmoqda',
  online: 'online',
  offline: 'offline'
}
const STATE_CLASS: Record<ClusterState, string> = {
  checking: 'text-faint',
  online: 'text-accent',
  offline: 'text-danger'
}

const MONTHS = ['yanvar', 'fevral', 'mart', 'aprel', 'may', 'iyun', 'iyul', 'avgust', 'sentabr', 'oktabr', 'noyabr', 'dekabr']
const d = new Date()
const today = `${d.getDate()}-${MONTHS[d.getMonth()]} ${d.getFullYear()}`

onMounted(async () => {
  if (requested) return
  requested = true
  try {
    const res = await api.health()
    health.state = 'online'
    health.version = res.version ?? ''
  } catch {
    health.state = 'offline'
  }
})
</script>
