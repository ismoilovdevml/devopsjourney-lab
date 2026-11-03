<!--
  LabMarkdown — renders task markdown (README) as sanitized HTML (utils/markdown: marked + DOMPurify).
  Props:
    source: string   (untrusted markdown)
-->
<template>
  <!-- eslint-disable-next-line vue/no-v-html -- sanitized by renderMarkdown (DOMPurify) -->
  <div class="lab-md" v-html="html" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{ source: string }>()

const html = computed(() => renderMarkdown(props.source))
</script>

<style scoped>
.lab-md {
  color: #c9d4e0;
  font-size: 14px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.lab-md > :deep(* + *) {
  margin-top: 12px;
}

.lab-md :deep(h1),
.lab-md :deep(h2),
.lab-md :deep(h3),
.lab-md :deep(h4) {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-weight: 600;
  color: #e6edf3;
  line-height: 1.35;
}

.lab-md :deep(h1),
.lab-md :deep(h2) {
  margin-top: 22px;
  font-size: 12.5px;
  color: #94a3b8;
}

.lab-md :deep(h1)::before,
.lab-md :deep(h2)::before {
  content: '# ';
}

.lab-md :deep(h3),
.lab-md :deep(h4) {
  margin-top: 18px;
  font-size: 13.5px;
}

.lab-md > :deep(:first-child) {
  margin-top: 0;
}

.lab-md :deep(ul),
.lab-md :deep(ol) {
  padding-left: 20px;
}

.lab-md :deep(ul) {
  list-style: disc;
}

.lab-md :deep(ol) {
  list-style: decimal;
}

.lab-md :deep(li + li),
.lab-md :deep(li > ul),
.lab-md :deep(li > ol) {
  margin-top: 4px;
}

.lab-md :deep(li::marker) {
  color: #3ddc97;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

.lab-md :deep(strong) {
  color: #e6edf3;
  font-weight: 600;
}

.lab-md :deep(a) {
  color: #3ddc97;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.lab-md :deep(a:hover) {
  color: #7ef0be;
}

.lab-md :deep(code) {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12.5px;
  color: #a78bfa;
  background: #0b1018;
  border: 1px solid #1a2332;
  border-radius: 5px;
  padding: 1px 5px;
}

.lab-md :deep(pre) {
  margin-top: 10px;
  padding: 12px 14px;
  background: #070a0f;
  border: 1px solid #1f2a3a;
  border-radius: 10px;
  overflow-x: auto;
}

.lab-md :deep(pre code) {
  display: block;
  padding: 0;
  border: 0;
  background: transparent;
  color: #3ddc97;
  line-height: 1.6;
  white-space: pre;
}

.lab-md :deep(blockquote) {
  padding: 10px 14px;
  background: rgba(56, 189, 248, 0.07);
  border: 1px solid rgba(56, 189, 248, 0.28);
  border-radius: 10px;
  color: #c9d4e0;
  font-size: 13px;
}

.lab-md :deep(blockquote > * + *) {
  margin-top: 6px;
}

.lab-md :deep(table) {
  display: block;
  width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
  font-size: 13px;
}

.lab-md :deep(th),
.lab-md :deep(td) {
  padding: 6px 10px;
  border: 1px solid #1f2a3a;
  text-align: left;
}

.lab-md :deep(th) {
  background: #0b1018;
  color: #94a3b8;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12px;
  font-weight: 500;
}

.lab-md :deep(hr) {
  border: 0;
  border-top: 1px solid #1a2332;
}
</style>
