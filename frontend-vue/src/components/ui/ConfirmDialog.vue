<!--
  ConfirmDialog — yes/no confirmation built on Modal (destructive actions: stop session, delete user,
  reset progress). Focus starts on the cancel button.
  Props:
    open: boolean            (v-model:open)
    title: string
    message?: string
    confirmLabel?: string    (default 'Tasdiqlash')
    cancelLabel?: string     (default 'Bekor qilish')
    variant?: 'danger' | 'primary'  (confirm button style; default 'danger')
    loading?: boolean        (spinner on confirm; dialog cannot be dismissed while loading)
  Emits: confirm, cancel, update:open
  The parent closes the dialog (set open=false) after the confirmed action finishes.
  Slots: default (extra body content under the message).
-->
<template>
  <Modal
    :open="open"
    :title="title"
    size="sm"
    :dismissible="!loading"
    @update:open="onUpdate"
  >
    <p v-if="message" class="text-sm text-text-2">{{ message }}</p>
    <slot />
    <template #footer>
      <Button variant="secondary" :disabled="loading" autofocus @click="cancel">{{ cancelLabel }}</Button>
      <Button :variant="variant" :loading="loading" @click="emit('confirm')">{{ confirmLabel }}</Button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import Modal from './Modal.vue'
import Button from './Button.vue'

withDefaults(
  defineProps<{
    open: boolean
    title: string
    message?: string
    confirmLabel?: string
    cancelLabel?: string
    variant?: 'danger' | 'primary'
    loading?: boolean
  }>(),
  { confirmLabel: 'Tasdiqlash', cancelLabel: 'Bekor qilish', variant: 'danger' }
)

const emit = defineEmits<{ confirm: []; cancel: []; 'update:open': [value: boolean] }>()

function cancel() {
  emit('cancel')
  emit('update:open', false)
}

function onUpdate(value: boolean) {
  if (!value) cancel()
}
</script>
