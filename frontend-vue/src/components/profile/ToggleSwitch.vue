<!--
  ToggleSwitch — accessible on/off switch (role="switch" + aria-checked).
  Props:
    modelValue: boolean   (v-model)
    label: string         (accessible name)
    disabled?: boolean
    describedby?: string  (id of a description element)
  Emits: update:modelValue(boolean)
-->
<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :aria-label="label"
    :aria-describedby="describedby"
    :disabled="disabled"
    :class="[
      'flex h-[26px] w-[46px] shrink-0 rounded-full p-[3px] transition-colors',
      modelValue ? 'justify-end bg-accent' : 'justify-start bg-border-strong',
      disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'
    ]"
    @click="emit('update:modelValue', !modelValue)"
  >
    <span
      :class="['block h-5 w-5 rounded-full', modelValue ? 'bg-accent-ink' : 'bg-muted']"
      aria-hidden="true"
    />
  </button>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: boolean
  label: string
  disabled?: boolean
  describedby?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>
