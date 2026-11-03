<!--
  TaskReadme — collapsible "README.md" block; markdown is rendered + sanitized by utils/markdown.
  Props:
    markdown: string
    defaultOpen?: boolean   (default false)
-->
<template>
  <div class="overflow-hidden rounded-btn border border-border bg-surface-2">
    <h3>
      <button
        :id="buttonId"
        type="button"
        class="flex w-full items-center gap-2 px-3.5 py-3 text-left font-mono text-[13px] font-semibold text-muted transition-colors hover:text-text"
        :aria-expanded="open"
        :aria-controls="panelId"
        @click="open = !open"
      >
        <ChevronRight :class="['h-4 w-4 shrink-0 transition-transform', open ? 'rotate-90' : '']" aria-hidden="true" />
        <span class="flex-1">README.md</span>
        <span class="text-xs font-normal text-faint">{{ open ? 'yopish' : "to'liq o'qish" }}</span>
      </button>
    </h3>
    <div
      v-show="open"
      :id="panelId"
      role="region"
      :aria-labelledby="buttonId"
      class="border-t border-divider px-4 py-3.5"
    >
      <!-- eslint-disable-next-line vue/no-v-html -- sanitized by DOMPurify in renderMarkdown -->
      <div class="readme" v-html="html" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronRight } from 'lucide-vue-next'
import { uid } from '@/components/ui/utils'
import { renderMarkdown } from '@/utils/markdown'

const props = withDefaults(defineProps<{ markdown: string; defaultOpen?: boolean }>(), { defaultOpen: false })

const open = ref(props.defaultOpen)
const buttonId = uid('readme-btn')
const panelId = uid('readme')
const html = computed(() => renderMarkdown(props.markdown))
</script>

<style scoped>
.readme {
  color: #c9d4e0;
  font-size: 13.5px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.readme > :deep(* + *) {
  margin-top: 0.75em;
}

.readme :deep(h1),
.readme :deep(h2),
.readme :deep(h3),
.readme :deep(h4) {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  color: #e6edf3;
  font-weight: 600;
  line-height: 1.35;
  margin-top: 1.1em;
}

.readme :deep(h1) {
  font-size: 16px;
}

.readme :deep(h2) {
  font-size: 14.5px;
}

.readme :deep(h3),
.readme :deep(h4) {
  font-size: 13.5px;
}

.readme > :deep(:first-child) {
  margin-top: 0;
}

.readme :deep(ul),
.readme :deep(ol) {
  padding-left: 1.25em;
}

.readme :deep(ul) {
  list-style: disc;
}

.readme :deep(ol) {
  list-style: decimal;
}

.readme :deep(li + li) {
  margin-top: 0.25em;
}

.readme :deep(a) {
  color: #3ddc97;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.readme :deep(a:hover) {
  color: #7ef0be;
}

.readme :deep(code) {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12.5px;
  background: #141c2a;
  border: 1px solid #1f2a3a;
  border-radius: 5px;
  padding: 0.05em 0.35em;
  color: #e6edf3;
}

.readme :deep(pre) {
  background: #070a0f;
  border: 1px solid #1f2a3a;
  border-radius: 9px;
  padding: 10px 12px;
  overflow-x: auto;
}

.readme :deep(pre code) {
  background: transparent;
  border: 0;
  padding: 0;
  color: #c9f5df;
  overflow-wrap: normal;
  white-space: pre;
}

.readme :deep(blockquote) {
  border-left: 2px solid #2a3850;
  padding-left: 12px;
  color: #94a3b8;
}

.readme :deep(table) {
  display: block;
  overflow-x: auto;
  border-collapse: collapse;
  font-size: 12.5px;
}

.readme :deep(th),
.readme :deep(td) {
  border: 1px solid #1f2a3a;
  padding: 4px 8px;
  text-align: left;
}

.readme :deep(hr) {
  border: 0;
  border-top: 1px solid #1a2332;
}
</style>
