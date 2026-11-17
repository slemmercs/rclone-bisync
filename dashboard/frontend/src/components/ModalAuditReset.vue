<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { useAuditStore } from '../stores/audit'

const auditStore = useAuditStore()
const resetting = ref(false)
const error = ref<string | null>(null)
const backupArchive = ref<string | null>(null)

function handleClose() {
  if (resetting.value) return
  auditStore.resetModalOpen = false
  error.value = null
  backupArchive.value = null
}

async function handleConfirmReset() {
  resetting.value = true
  error.value = null
  try {
    const res = await auditStore.resetAuditLedger()
    if (res.backup) {
      backupArchive.value = res.backup
    }
    setTimeout(() => {
      handleClose()
    }, 1500)
  } catch (err: any) {
    error.value = err.message || 'Failed to reset audit ledger'
  } finally {
    resetting.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="auditStore.resetModalOpen"
    @update:visible="(val) => { if (!resetting) auditStore.resetModalOpen = val }"
    modal
    :style="{ width: '32rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-rose-500/30 rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-4' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 shrink-0">
          <i class="pi pi-exclamation-triangle text-base"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white">Reset Immutable Audit Ledger</h3>
          <div class="text-[11px] text-slate-400">Preserves timestamped backup archive</div>
        </div>
      </div>
    </template>

    <div class="space-y-4">
      <div class="p-3 rounded-xl bg-rose-950/30 border border-rose-800/40 text-xs text-rose-200 leading-relaxed font-sans">
        Are you sure you want to reset the audit ledger? The entire blockchain-style SHA-256 hash chain will be preserved in a timestamped backup before a fresh genesis block is initialized.
      </div>

      <div v-if="backupArchive" class="p-3 rounded-xl bg-emerald-950/60 border border-emerald-800/60 text-emerald-300 text-xs font-mono">
        <i class="pi pi-check text-emerald-400 mr-1.5"></i>
        <span>Ledger reset. Backup preserved: {{ backupArchive }}</span>
      </div>

      <div v-if="error" class="p-3 rounded-xl bg-rose-950/60 border border-rose-800/60 text-rose-300 text-xs font-mono">
        <i class="pi pi-times-circle text-rose-400 mr-1.5"></i>
        <span>{{ error }}</span>
      </div>

      <div class="pt-3 border-t border-carbon-border flex items-center justify-between">
        <span class="text-[10px] text-slate-500 font-mono">Blockchain continuity reset</span>
        <div class="flex items-center gap-2">
          <button
            type="button"
            @click="handleClose"
            :disabled="resetting"
            class="px-4 py-2 bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border rounded-xl text-xs font-semibold transition-colors disabled:opacity-50 cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            @click="handleConfirmReset"
            :disabled="resetting"
            class="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl text-xs font-bold font-mono transition-colors flex items-center gap-2 disabled:opacity-50 cursor-pointer shadow"
          >
            <i v-if="resetting" class="pi pi-spin pi-spinner text-xs"></i>
            <span>{{ resetting ? 'Resetting...' : 'Reset Ledger' }}</span>
          </button>
        </div>
      </div>
    </div>
  </Dialog>
</template>
