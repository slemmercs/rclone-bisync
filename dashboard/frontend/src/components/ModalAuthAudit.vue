<script setup lang="ts">
import Dialog from 'primevue/dialog'
import { useRemotesStore } from '../stores/remotes'

const remotesStore = useRemotesStore()

function handleClose() {
  remotesStore.auditModal.show = false
}

function formatDate(isoStr?: string): string {
  if (!isoStr) return '--'
  try {
    const d = new Date(isoStr)
    return d.toLocaleString()
  } catch {
    return isoStr
  }
}
</script>

<template>
  <Dialog
    :visible="remotesStore.auditModal.show"
    @update:visible="(val) => remotesStore.auditModal.show = val"
    modal
    :style="{ width: '42rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-3 max-h-[80vh] overflow-y-auto' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-electric-blue/10 border border-electric-blue/20 text-electric-blue shrink-0">
          <i class="pi pi-history text-base"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white">OAuth Authentication Audit Log</h3>
          <div class="text-[11px] text-slate-400">Recorded authentication events and credential updates</div>
        </div>
      </div>
    </template>

    <div class="space-y-2">
      <!-- Loading State -->
      <div v-if="remotesStore.auditModal.loading" class="text-electric-blue animate-pulse text-center py-8 text-xs font-mono">
        <i class="pi pi-spin pi-spinner mr-2"></i>
        <span>Loading authentication audit events...</span>
      </div>

      <!-- Events List -->
      <div v-else-if="remotesStore.auditModal.events.length > 0" class="space-y-2 font-mono text-xs">
        <div
          v-for="(evt, idx) in remotesStore.auditModal.events"
          :key="idx"
          class="p-3 rounded-xl bg-carbon-bg border border-carbon-border flex items-start justify-between gap-3 text-[11px]"
        >
          <div class="space-y-1">
            <div class="flex items-center gap-2">
              <span class="font-bold text-slate-200">{{ evt.event }}</span>
              <span class="px-1.5 py-0.5 rounded font-mono text-[10px] bg-carbon-elevated border border-carbon-border text-electric-blue">
                {{ evt.remote }}
              </span>
              <span
                class="px-1.5 py-0.5 rounded font-mono text-[9px] uppercase font-bold"
                :class="
                  evt.status === 'success'
                    ? 'bg-emerald-950 text-emerald-400 border border-emerald-800/40'
                    : evt.status === 'failed' || evt.status === 'error'
                    ? 'bg-rose-950 text-rose-400 border border-rose-800/40'
                    : 'bg-carbon-surface text-slate-400 border border-carbon-border'
                "
              >
                {{ evt.status }}
              </span>
            </div>
            <p v-if="evt.details" class="text-slate-400 font-sans text-xs">
              {{ evt.details }}
            </p>
          </div>
          <span class="text-slate-500 text-[10px] shrink-0" :title="evt.timestamp">
            {{ formatDate(evt.timestamp) }}
          </span>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="text-slate-500 text-center py-8 text-xs font-mono">
        No authentication audit events recorded yet.
      </div>

      <!-- Footer -->
      <div class="pt-3 border-t border-carbon-border flex justify-end">
        <button
          type="button"
          @click="handleClose"
          class="px-4 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border rounded-xl text-xs font-semibold transition-colors cursor-pointer"
        >
          Close
        </button>
      </div>
    </div>
  </Dialog>
</template>
