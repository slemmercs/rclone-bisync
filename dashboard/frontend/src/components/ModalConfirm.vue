<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    :header="title || 'Confirm Destructive Action'"
    :style="{ width: '32rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-rose-500/30 rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between text-rose-400 font-bold' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <div class="space-y-3 text-xs">
      <div class="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl text-rose-300 flex items-start gap-2.5">
        <i class="pi pi-exclamation-triangle text-base text-rose-400 mt-0.5 shrink-0"></i>
        <div class="space-y-1">
          <p class="font-bold">{{ message }}</p>
          <p class="text-[11px] text-rose-400/80 font-mono break-all">{{ target }}</p>
        </div>
      </div>
      <p class="text-[11px] text-slate-400">
        This action is irreversible and permanently deletes data from the target storage system.
      </p>
    </div>

    <div class="flex items-center justify-end gap-2 border-t border-carbon-border pt-3">
      <button
        @click="emit('update:visible', false)"
        class="px-3 py-1.5 bg-carbon-elevated text-slate-300 hover:bg-slate-800 rounded-xl text-xs cursor-pointer"
      >
        Cancel
      </button>
      <button
        @click="emit('confirm')"
        :disabled="loading"
        class="px-4 py-1.5 bg-rose-600 hover:bg-rose-500 text-white rounded-xl text-xs font-semibold cursor-pointer disabled:opacity-50 flex items-center gap-1.5 shadow-lg shadow-rose-950"
      >
        <i v-if="loading" class="pi pi-spin pi-spinner text-xs"></i>
        <i v-else class="pi pi-trash text-xs"></i>
        <span>{{ loading ? 'Deleting...' : (confirmLabel || 'Delete Permanently') }}</span>
      </button>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import Dialog from 'primevue/dialog'

defineProps<{
  visible: boolean
  title?: string
  message: string
  target: string
  confirmLabel?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'confirm'): void
}>()
</script>
