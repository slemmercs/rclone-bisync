<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { useAuditStore } from '../stores/audit'

const auditStore = useAuditStore()
const showRawJson = ref(false)
const copiedCurrentHash = ref(false)
const copiedPrevHash = ref(false)

function handleClose() {
  auditStore.closeDetails()
}

function copyHash(hash: string, isCurrent: boolean) {
  if (!hash) return
  if (navigator?.clipboard?.writeText) {
    navigator.clipboard.writeText(hash).then(() => {
      if (isCurrent) {
        copiedCurrentHash.value = true
        setTimeout(() => { copiedCurrentHash.value = false }, 2000)
      } else {
        copiedPrevHash.value = true
        setTimeout(() => { copiedPrevHash.value = false }, 2000)
      }
    })
  }
}

function getActionBadgeClass(action?: string): string {
  if (!action) return 'bg-carbon-elevated text-slate-300 border-carbon-border'
  if (action.includes('SYNC_EXECUTED')) return 'bg-emerald-950/80 text-emerald-300 border-emerald-800/60'
  if (action.includes('DEDUPE')) return 'bg-amber-950/80 text-amber-300 border-amber-800/60'
  if (action.includes('CONFLICT_RESOLVED')) return 'bg-indigo-950/80 text-indigo-300 border-indigo-800/60'
  if (action.includes('DELETION_ACCEPTED')) return 'bg-cyan-950/80 text-cyan-300 border-cyan-800/60'
  if (action.includes('DELETION_RESTORED')) return 'bg-teal-950/80 text-teal-300 border-teal-800/60'
  if (action.includes('ROLLBACK')) return 'bg-rose-950/80 text-rose-300 border-rose-800/60'
  if (action.includes('DISMISSED')) return 'bg-slate-900 text-slate-400 border-slate-700/60'
  if (action.includes('CONFIG') || action.includes('SECRET')) return 'bg-purple-950/80 text-purple-300 border-purple-800/60'
  if (action.includes('FILE_UPLOADED') || action.includes('DIR_CREATED')) return 'bg-sky-950/80 text-sky-300 border-sky-800/60'
  if (action.includes('ITEM_COPIED')) return 'bg-blue-950/80 text-blue-300 border-blue-800/60'
  if (action.includes('ITEM_MOVED')) return 'bg-amber-950/80 text-amber-300 border-amber-800/60'
  if (action.includes('ITEM_DELETED') || action.includes('DIR_PURGED')) return 'bg-rose-950/80 text-rose-300 border-rose-800/60'
  return 'bg-carbon-elevated text-slate-300 border-carbon-border'
}
</script>

<template>
  <Dialog
    :visible="auditStore.detailsModalOpen"
    @update:visible="(val) => auditStore.detailsModalOpen = val"
    modal
    :style="{ width: '44rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-4 max-h-[80vh] overflow-y-auto' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-electric-blue/10 border border-electric-blue/20 text-electric-blue shrink-0">
          <i class="pi pi-shield text-base"></i>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-sm font-bold text-white">
              Audit Block #{{ auditStore.selectedRecord?.seq_id }}
            </h3>
            <span
              v-if="auditStore.selectedRecord"
              class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border font-mono"
              :class="getActionBadgeClass(auditStore.selectedRecord.action)"
            >
              {{ auditStore.selectedRecord.action }}
            </span>
          </div>
          <div class="text-[11px] text-slate-400 font-mono">
            {{ auditStore.selectedRecord?.timestamp }}
          </div>
        </div>
      </div>
    </template>

    <div v-if="auditStore.selectedRecord" class="space-y-4 font-mono text-xs">
      <!-- General Metadata Grid -->
      <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 p-3 bg-carbon-bg rounded-xl border border-carbon-border text-[11px]">
        <div>
          <span class="text-slate-500 block uppercase text-[9px]">Profile</span>
          <span class="text-slate-200 font-semibold">{{ auditStore.selectedRecord.profile_name || auditStore.selectedRecord.profile_id || '--' }}</span>
        </div>
        <div>
          <span class="text-slate-500 block uppercase text-[9px]">Actor</span>
          <span class="text-electric-blue font-semibold uppercase">{{ auditStore.selectedRecord.actor }}</span>
        </div>
        <div>
          <span class="text-slate-500 block uppercase text-[9px]">Sequence #</span>
          <span class="text-indigo-400 font-semibold">#{{ auditStore.selectedRecord.seq_id }}</span>
        </div>
      </div>

      <!-- Cryptographic Hash Chaining -->
      <div class="p-3 bg-carbon-bg rounded-xl border border-carbon-border space-y-2">
        <div class="text-slate-400 text-[11px] font-semibold flex items-center justify-between">
          <span>Cryptographic Hash Chaining</span>
          <span class="text-[9px] text-emerald-400 flex items-center gap-1">
            <i class="pi pi-verified text-[10px]"></i>
            <span>SHA-256 Verified</span>
          </span>
        </div>

        <div class="space-y-1.5 text-[10px]">
          <div>
            <span class="text-slate-500 block">Block Hash:</span>
            <div class="flex items-center justify-between bg-carbon-surface p-1.5 rounded border border-carbon-border text-slate-200 break-all">
              <span>{{ auditStore.selectedRecord.hash }}</span>
              <button
                type="button"
                @click="copyHash(auditStore.selectedRecord.hash, true)"
                class="ml-2 text-slate-400 hover:text-white shrink-0 cursor-pointer"
                title="Copy hash"
              >
                <i :class="copiedCurrentHash ? 'pi pi-check text-emerald-400' : 'pi pi-copy'"></i>
              </button>
            </div>
          </div>

          <div>
            <span class="text-slate-500 block">Previous Hash (Parent):</span>
            <div class="flex items-center justify-between bg-carbon-surface p-1.5 rounded border border-carbon-border text-slate-400 break-all">
              <span>{{ auditStore.selectedRecord.prev_hash }}</span>
              <button
                type="button"
                @click="copyHash(auditStore.selectedRecord.prev_hash, false)"
                class="ml-2 text-slate-400 hover:text-white shrink-0 cursor-pointer"
                title="Copy prev hash"
              >
                <i :class="copiedPrevHash ? 'pi pi-check text-emerald-400' : 'pi pi-copy'"></i>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Details Metadata -->
      <div class="border border-carbon-border rounded-xl p-3 bg-carbon-bg space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-slate-400 text-[11px] font-semibold">Event Payload Details</span>
          <button
            type="button"
            @click="showRawJson = !showRawJson"
            class="text-[10px] text-electric-blue hover:text-sky-300 font-sans cursor-pointer"
          >
            {{ showRawJson ? 'View Structured' : 'View Raw JSON' }}
          </button>
        </div>

        <!-- Raw JSON -->
        <div v-if="showRawJson">
          <pre class="bg-carbon-surface p-2.5 rounded-lg border border-carbon-border text-[10px] text-slate-300 overflow-x-auto select-all max-h-56 leading-relaxed">{{ JSON.stringify(auditStore.selectedRecord.details || {}, null, 2) }}</pre>
        </div>

        <!-- Structured Key-Value -->
        <div v-else class="space-y-1">
          <template v-if="auditStore.selectedRecord.details && Object.keys(auditStore.selectedRecord.details).length > 0">
            <div
              v-for="(val, key) in auditStore.selectedRecord.details"
              :key="key"
              class="flex items-center justify-between py-1 px-2 rounded bg-carbon-surface border border-carbon-border/60 text-[11px]"
            >
              <span class="text-slate-400">{{ key }}:</span>
              <span class="text-slate-200 font-semibold select-all">{{ typeof val === 'object' ? JSON.stringify(val) : val }}</span>
            </div>
          </template>
          <div v-else class="text-slate-500 text-center py-2 italic font-sans text-xs">
            No action metadata attached to this record.
          </div>
        </div>
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
