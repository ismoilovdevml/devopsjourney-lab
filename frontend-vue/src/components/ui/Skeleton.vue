<!--
  Skeleton — shimmering placeholder while data loads (aria-hidden).
  Props:
    width?: string     (CSS width, default '100%')
    height?: string    (CSS height, default '14px')
    rounded?: string   (Tailwind radius class, default 'rounded-md')
    lines?: number     (render n stacked text lines; last one is 60% wide)
-->
<template>
  <span v-if="lines && lines > 1" class="flex w-full flex-col gap-2" aria-hidden="true">
    <span
      v-for="i in lines"
      :key="i"
      :class="['skeleton block', rounded]"
      :style="{ width: i === lines ? '60%' : width, height }"
    />
  </span>
  <span v-else :class="['skeleton block', rounded]" :style="{ width, height }" aria-hidden="true" />
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    width?: string
    height?: string
    rounded?: string
    lines?: number
  }>(),
  { width: '100%', height: '14px', rounded: 'rounded-md' }
)
</script>

<style scoped>
.skeleton {
  background: linear-gradient(90deg, #141c2a 0%, #1b2535 50%, #141c2a 100%);
  background-size: 200% 100%;
  animation: shimmer 1.6s linear infinite;
}

@keyframes shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}
</style>
