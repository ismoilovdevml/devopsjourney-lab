<!--
  ResetProgressDialog — destructive confirmation for DELETE /api/progress. The user must type their
  username before the confirm button is enabled. Performs the request itself.
  Built on Modal (not ConfirmDialog) because ConfirmDialog has no way to disable its confirm button.
  Props:
    open: boolean        (v-model:open)
    username: string     (value that must be typed to confirm)
  Emits: update:open, reset (after the server confirmed the reset)
-->
<template>
  <Modal
    :open="open"
    title="Progressni tiklash"
    size="sm"
    :dismissible="!busy"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <form :id="formId" class="flex flex-col gap-4" @submit.prevent="submit">
      <p class="text-sm text-text-2">
        Barcha vazifalar bo'yicha natijalar, urinishlar va yutuqlar o'chiriladi. Bu amalni ortga qaytarib
        bo'lmaydi.
      </p>
      <p class="text-[13px] text-muted">
        Laboratoriyada o'tkazilgan vaqt (sessiyalar tarixi) saqlanib qoladi.
      </p>
      <div>
        <label :for="inputId" class="label">
          Tasdiqlash uchun foydalanuvchi nomingizni kiriting:
          <span class="font-mono text-text">{{ username }}</span>
        </label>
        <input
          :id="inputId"
          v-model="typed"
          type="text"
          autofocus
          class="input font-mono"
          autocomplete="off"
          autocapitalize="off"
          spellcheck="false"
          :disabled="busy"
          :aria-invalid="error ? 'true' : undefined"
          :aria-describedby="error ? errorId : undefined"
        />
      </div>
      <p v-if="error" :id="errorId" role="alert" class="text-[13px] text-danger">{{ error }}</p>
    </form>
    <template #footer>
      <Button variant="secondary" :disabled="busy" @click="emit('update:open', false)">Bekor qilish</Button>
      <Button variant="danger" type="submit" :form="formId" :loading="busy" :disabled="!matches" :icon="RotateCcw">
        Progressni tiklash
      </Button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RotateCcw } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { uid } from '@/components/ui/utils'
import { api, errorMessage } from '@/services/api'

const props = defineProps<{
  open: boolean
  username: string
}>()

const emit = defineEmits<{ 'update:open': [value: boolean]; reset: [] }>()

const formId = uid('reset-form')
const inputId = uid('reset-input')
const errorId = uid('reset-error')

const typed = ref('')
const busy = ref(false)
const error = ref<string | null>(null)

const matches = computed(() => props.username.length > 0 && typed.value.trim() === props.username)

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      typed.value = ''
      error.value = null
    }
  }
)

async function submit() {
  if (!matches.value || busy.value) return
  busy.value = true
  error.value = null
  try {
    await api.resetProgress()
    emit('reset')
    emit('update:open', false)
  } catch (err) {
    error.value = errorMessage(err, "Progressni tiklab bo'lmadi. Qayta urinib ko'ring.")
  } finally {
    busy.value = false
  }
}
</script>
