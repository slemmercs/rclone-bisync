<script setup lang="ts">
import Dialog from 'primevue/dialog'
import { useExplorerStore } from '../stores/explorer'
import { useProfilesStore } from '../stores/profiles'

const explorerStore = useExplorerStore()
const profilesStore = useProfilesStore()

function handleClose() {
  explorerStore.closeUploadDockModal()
}

function handleRefresh() {
  explorerStore.fetchUploadDockStatus(profilesStore.activeProfile?.id)
}

function handleClearCompleted() {
  explorerStore.clearCompletedUploads(profilesStore.activeProfile?.id)
}

function handleRemoveItem(id: string) {
  explorerStore.removeDockItem(id, profilesStore.activeProfile?.id)
}

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null || isNaN(bytes)) return '0 B'
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

function getStatusBadgeClass(status: string): string {
  switch (status) {
    case 'queued':
      return 'bg-carbon-elevated text-slate-400 border border-carbon-border'
    case 'waiting_for_sync':
      return 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
    case 'uploading':
      return 'bg-sky-500/20 text-sky-300 border border-sky-500/30 animate-pulse'
    case 'completed':
      return 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
    case 'failed':
      return 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
    default:
      return 'bg-carbon-elevated text-slate-400 border border-carbon-border'
  }
}
</script>

<template>
  <Dialog
    :visible="explorerStore.uploadDockModalOpen"
    @update:visible="(val) => explorerStore.uploadDockModalOpen = val"
    modal
    :style="{ width: '42rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-4' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-electric-blue/10 border border-electric-blue/20 text-electric-blue shrink-0">
          <i class="pi pi-upload text-base"></i>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-sm font-bold text-white">Upload Dock & Transfer Queue</h3>
            <span
              v-if="explorerStore.uploadDock.is_sync_active"
              class="px-2 py-0.5 rounded text-[10px] bg-amber-500/20 text-amber-300 border border-amber-500/30 uppercase tracking-wider font-bold"
            >
              Sync In Progress (Deferred)
            </span>
          </div>
          <div class="text-[11px] text-slate-400">
            Real-time multi-threaded upload pipeline
          </div>
        </div>
      </div>
    </template>

    <div class="space-y-3 font-mono text-xs">
      <div class="flex items-center justify-between text-slate-400 text-xs">
        <span>Pending Items: {{ explorerStore.uploadDock.pending_count || 0 }}</span>
        <div class="flex items-center gap-3 font-sans">
          <button
            type="button"
            @click="handleRefresh"
            class="hover:text-electric-blue transition-colors flex items-center gap-1 cursor-pointer"
          >
            <i :class="explorerStore.uploadDockLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
            <span>Refresh</span>
          </button>
          <button
            type="button"
            @click="handleClearCompleted"
            class="text-rose-400 hover:text-rose-300 transition-colors cursor-pointer"
          >
            Clear Completed
          </button>
        </div>
      </div>

      <!-- Items Table -->
      <div class="max-h-72 overflow-y-auto border border-carbon-border rounded-xl bg-carbon-bg divide-y divide-carbon-border">
        <div
          v-if="!explorerStore.uploadDock.items || explorerStore.uploadDock.items.length === 0"
          class="p-8 text-center text-slate-500 text-xs font-sans"
        >
          No uploads in dock queue.
        </div>

        <div
          v-for="item in explorerStore.uploadDock.items"
          :key="item.id"
          class="p-3 flex items-center justify-between gap-3 text-xs"
        >
          <div class="min-w-0 flex-1">
            <div class="font-medium text-slate-200 truncate font-mono">
              {{ item.file_name }}
            </div>
            <div class="text-[10px] text-slate-400 truncate font-mono">
              {{ item.remote }}:{{ item.path || '/' }} • {{ formatBytes(item.size) }}
            </div>
            <div v-if="item.error" class="text-[10px] text-rose-400 truncate mt-0.5 font-sans">
              Error: {{ item.error }}
            </div>
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <span
              class="px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase tracking-wider"
              :class="getStatusBadgeClass(item.status)"
            >
              {{ item.status }}
            </span>
            <button
              type="button"
              @click="handleRemoveItem(item.id)"
              class="text-slate-500 hover:text-rose-400 p-1 rounded transition-colors cursor-pointer"
              title="Remove item"
            >
              <i class="pi pi-times text-xs"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="p-2.5 rounded-lg bg-carbon-bg border border-carbon-border text-[11px] text-slate-400 font-sans leading-relaxed">
        <span class="text-electric-blue font-semibold">Non-Blocking Coordination:</span> When profile synchronization is active, uploads are held safely in a waiting state to prevent corruption, and resumed automatically once locks release.
      </div>

      <div class="flex items-center justify-end pt-2 border-t border-carbon-border">
        <button
          type="button"
          @click="handleClose"
          class="px-4 py-2 bg-carbon-elevated hover:bg-slate-800 text-slate-300 rounded-xl text-xs font-medium transition-colors cursor-pointer border border-carbon-border"
        >
          Close
        </button>
      </div>
    </div>
  </Dialog>
</template>
