<template>
  <Dialog
    :visible="dedupeStore.modalOpen"
    @update:visible="val => !val && dedupeStore.closeModal()"
    modal
    :style="{ width: '95vw', maxWidth: '1200px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl shadow-2xl overflow-hidden' },
      header: { class: 'p-4 sm:p-5 border-b border-carbon-border bg-carbon-surface flex justify-between items-center text-white' },
      content: { class: 'p-0 bg-carbon-canvas flex flex-col h-[75vh]' },
      closeButton: { class: 'text-slate-400 hover:text-white hover:bg-carbon-elevated rounded-lg p-1.5 transition-colors cursor-pointer' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-3 min-w-0">
        <div class="p-2.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-400 shrink-0">
          <i class="pi pi-clone text-base"></i>
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <h3 class="text-sm font-bold text-white tracking-wide uppercase">Deduplication Workspace</h3>
            <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-amber-950/80 text-amber-300 border border-amber-700/60 font-semibold">
              {{ dedupeStore.remote }}:{{ dedupeStore.path || '/' }}
            </span>
          </div>
          <div class="text-[11px] text-slate-400 truncate">
            Inspect duplicate files, review collision candidates, and purge redundant copies
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2 mr-2">
        <span
          v-if="dedupeStore.cachedTimestamp"
          class="px-2 py-0.5 rounded bg-indigo-950/40 border border-indigo-700/50 text-indigo-300 font-mono text-[11px] flex items-center gap-1.5"
          title="Scan results restored from session cache"
        >
          <i class="pi pi-database text-[10px]"></i>
          <span>Cached ({{ formatRelativeTime(dedupeStore.cachedTimestamp) }})</span>
          <button
            type="button"
            @click="dedupeStore.clearCache()"
            class="text-slate-400 hover:text-rose-400 ml-0.5 cursor-pointer"
            title="Clear cached scan results"
          >
            <i class="pi pi-times text-[10px]"></i>
          </button>
        </span>
        <span class="px-2 py-0.5 rounded bg-carbon-elevated text-slate-300 font-mono text-[11px] border border-carbon-border">
          Collisions: {{ dedupeStore.parsed.totalCollisions }}
        </span>
        <span
          v-if="dedupeStore.parsed.totalDeleted > 0"
          class="px-2 py-0.5 rounded bg-rose-950/40 border border-rose-800/40 text-rose-300 font-mono text-[11px]"
        >
          Deleted: {{ dedupeStore.parsed.totalDeleted }}
        </span>
      </div>
    </template>

    <!-- Workspace Body -->
    <div class="flex-1 flex flex-col md:flex-row overflow-hidden divide-y md:divide-y-0 md:divide-x divide-carbon-border">
      <!-- Settings Rail -->
      <div class="w-full md:w-80 shrink-0 flex flex-col bg-carbon-surface/40 p-4 overflow-y-auto space-y-4">
        <div class="space-y-3">
          <!-- Dedupe Mode -->
          <div>
            <label class="block text-xs font-semibold text-slate-300 mb-1">Deduplication Strategy</label>
            <select
              v-model="dedupeStore.mode"
              class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-xs text-slate-200 font-mono focus:border-amber-500 cursor-pointer"
            >
              <option value="list">list — Report duplicates without deleting</option>
              <option value="oldest">oldest — Keep the oldest duplicate</option>
              <option value="newest">newest — Keep the newest duplicate</option>
              <option value="rename">rename — Rename identical files uniquely</option>
              <option value="largest">largest — Keep the largest file copy</option>
              <option value="smallest">smallest — Keep the smallest file copy</option>
            </select>
          </div>

          <!-- Match Strategy Dropdown -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="block text-xs font-semibold text-slate-300">Match Strategy</label>
              <span
                class="text-[10px] font-mono px-1.5 py-0.2 rounded"
                :class="dedupeStore.byHash ? 'bg-indigo-950/80 text-indigo-300 border border-indigo-700/60' : 'bg-emerald-950/80 text-emerald-300 border border-emerald-700/60'"
              >
                {{ dedupeStore.byHash ? '--by-hash' : 'same-folder' }}
              </span>
            </div>
            <select
              :value="dedupeStore.byHash ? 'hash' : 'name'"
              @change="dedupeStore.byHash = (($event.target as HTMLSelectElement).value === 'hash')"
              class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-xs text-slate-200 font-mono focus:border-amber-500 cursor-pointer"
            >
              <option value="name">Same-Folder Duplicate Names (Safe Default)</option>
              <option value="hash">Cross-Folder Hash (--by-hash)</option>
            </select>
            <p class="text-[10px] text-slate-400 mt-1">
              {{ dedupeStore.byHash
                  ? 'Finds identical duplicates across all folders by cryptographic checksum.'
                  : 'Resolves duplicate filenames within identical directories (Drive / multi-upload).' }}
            </p>
          </div>

          <!-- Target Subdirectory -->
          <div>
            <label class="block text-xs font-semibold text-slate-300 mb-1">Target Subdirectory</label>
            <input
              type="text"
              v-model="dedupeStore.path"
              placeholder="e.g. documents/archive (empty for all)"
              class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-xs text-slate-200 font-mono focus:border-amber-500"
            />
          </div>

          <!-- Paired Pass Toggle -->
          <div class="pt-1 border-t border-carbon-border/60">
            <label class="flex items-center gap-2 cursor-pointer select-none">
              <input
                type="checkbox"
                v-model="dedupeStore.dedupeBoth"
                class="rounded bg-carbon-elevated border-carbon-border text-amber-500 focus:ring-0 w-3.5 h-3.5"
              />
              <span class="text-xs text-slate-300 font-medium">Paired Pass (Remote + Local)</span>
            </label>
            <p class="text-[10px] text-slate-400 mt-0.5 ml-5.5">
              Deduplicate both local and remote endpoints of linked profile.
            </p>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="pt-3 border-t border-carbon-border space-y-2 mt-auto">
          <button
            type="button"
            @click="handleRun(true)"
            :disabled="dedupeStore.loading"
            class="w-full py-2 bg-amber-500 hover:bg-amber-400 disabled:opacity-50 text-black font-bold text-xs rounded-xl shadow transition-colors flex items-center justify-center gap-1.5 cursor-pointer"
          >
            <i class="pi" :class="dedupeStore.loading ? 'pi-spin pi-spinner' : 'pi-search'"></i>
            <span>Scan for Duplicates (Dry Run)</span>
          </button>

          <button
            type="button"
            @click="confirmExecute()"
            :disabled="dedupeStore.loading || dedupeStore.mode === 'list'"
            class="w-full py-2 bg-rose-600 hover:bg-rose-500 disabled:opacity-40 text-white font-bold text-xs rounded-xl transition-colors flex items-center justify-center gap-1.5 cursor-pointer"
            :title="dedupeStore.mode === 'list' ? 'Select a resolution mode above to execute' : 'Permanently resolve duplicate files on disk'"
          >
            <i class="pi pi-trash"></i>
            <span>Execute {{ dedupeStore.mode.toUpperCase() }} Resolution</span>
          </button>

          <!-- Correlated Daemon Modal Link -->
          <button
            v-if="dedupeStore.runId"
            type="button"
            @click="openDaemonLogs()"
            class="w-full py-1.5 bg-carbon-elevated hover:bg-slate-700 text-electric-300 border border-carbon-border text-[11px] font-mono rounded-xl transition-colors flex items-center justify-center gap-1.5 cursor-pointer"
          >
            <i class="pi pi-terminal text-xs"></i>
            <span>View Daemon Logs ({{ dedupeStore.runId.substring(0, 12) }}...)</span>
          </button>
        </div>
      </div>

      <!-- Results Center Panel -->
      <div class="flex-1 flex flex-col overflow-hidden bg-carbon-canvas relative">
        <!-- View Toggle & Status Bar -->
        <div class="p-3 bg-carbon-surface/60 border-b border-carbon-border flex items-center justify-between gap-3 text-xs">
          <div class="flex items-center gap-2">
            <button
              type="button"
              @click="activeTab = 'items'"
              :class="activeTab === 'items' ? 'bg-amber-500 text-black font-bold' : 'text-slate-400 hover:text-white'"
              class="px-2.5 py-1 rounded-lg text-xs transition-colors cursor-pointer"
            >
              Candidates ({{ dedupeStore.parsed.items.length }})
            </button>
            <button
              type="button"
              @click="switchTab('explorer')"
              :class="activeTab === 'explorer' ? 'bg-amber-500 text-black font-bold' : 'text-slate-400 hover:text-white'"
              class="px-2.5 py-1 rounded-lg text-xs transition-colors cursor-pointer flex items-center gap-1.5"
            >
              <i class="pi pi-folder text-xs"></i>
              <span>Explorer</span>
            </button>
            <button
              type="button"
              @click="activeTab = 'raw'"
              :class="activeTab === 'raw' ? 'bg-amber-500 text-black font-bold' : 'text-slate-400 hover:text-white'"
              class="px-2.5 py-1 rounded-lg text-xs transition-colors cursor-pointer"
            >
              Raw Output
            </button>
          </div>

          <span v-if="dedupeStore.loading" class="text-amber-400 flex items-center gap-1.5 text-xs font-mono">
            <i class="pi pi-spin pi-spinner"></i>
            <span>Running deduplication...</span>
          </span>
        </div>

        <!-- Notices & Errors -->
        <div v-if="dedupeStore.parsed.notices.length > 0" class="p-3 bg-amber-500/10 border-b border-amber-500/20 text-amber-300 text-xs flex items-center gap-2">
          <i class="pi pi-info-circle text-amber-400 shrink-0"></i>
          <span>{{ dedupeStore.parsed.notices.join(' • ') }}</span>
        </div>
        <div v-if="dedupeStore.parsed.errors.length > 0" class="p-3 bg-rose-500/10 border-b border-rose-500/20 text-rose-300 text-xs flex items-center gap-2">
          <i class="pi pi-exclamation-triangle text-rose-400 shrink-0"></i>
          <span>{{ dedupeStore.parsed.errors.join(' • ') }}</span>
        </div>

        <!-- Candidates List Tab -->
        <DedupeGroupCluster
          v-if="activeTab === 'items'"
          @switch-tab="switchTab"
        />

        <!-- Directory Explorer Tab -->
        <DedupeExplorerTab
          v-else-if="activeTab === 'explorer'"
          @preview="handlePreview"
        />

        <!-- Raw CLI Output Tab -->
        <div v-else-if="activeTab === 'raw'" class="flex-1 overflow-y-auto p-4 bg-carbon-canvas font-mono text-xs text-slate-300 select-text whitespace-pre-wrap leading-relaxed">
          {{ dedupeStore.rawOutput || 'No output recorded yet. Run a scan to see raw rclone CLI logs.' }}
        </div>

        <!-- Staged Actions Review Slide-Up Drawer -->
        <DedupeStagedDrawer />

        <!-- Staged Batch Decisions Bottom Bar -->
        <DedupeActionBar />
      </div>
    </div>

    <!-- File Preview Modal -->
    <ModalFilePreview
      v-model:visible="showPreviewModal"
      :preview="previewData"
      :loading="previewLoading"
      :error="previewError"
    />
  </Dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { api } from '../api/client'
import { useDedupeStore } from '../stores/dedupe'
import { useLogsStore } from '../stores/logs'
import { formatRelativeTime } from '../composables/useFormatters'
import { useToastNotification } from '../composables/useToastNotification'
import DedupeGroupCluster from './dedupe/DedupeGroupCluster.vue'
import DedupeActionBar from './dedupe/DedupeActionBar.vue'
import DedupeStagedDrawer from './dedupe/DedupeStagedDrawer.vue'
import DedupeExplorerTab from './dedupe/DedupeExplorerTab.vue'
import ModalFilePreview from './ModalFilePreview.vue'
import type { LogEntry } from '../types/logs'
import type { BrowseItem } from '../types/models'
import type { FilePreviewResponse } from '../types/explorer'

const dedupeStore = useDedupeStore()
const logsStore = useLogsStore()
const toast = useToastNotification()
const activeTab = ref<'items' | 'explorer' | 'raw'>('items')

// File Preview state
const showPreviewModal = ref(false)
const previewData = ref<FilePreviewResponse | null>(null)
const previewLoading = ref(false)
const previewError = ref<string | null>(null)

function switchTab(tab: 'items' | 'explorer' | 'raw') {
  activeTab.value = tab
  if (tab === 'explorer' && (!dedupeStore.explorerItems || dedupeStore.explorerItems.length === 0)) {
    dedupeStore.loadExplorer()
  }
}

function handleRun(dryRun: boolean) {
  dedupeStore.runDedupe(dryRun)
  activeTab.value = 'items'
  toast.info(`Scanning duplicates (${dryRun ? 'Dry Run' : 'Live'})...`, 'Dedupe Scan')
}

function confirmExecute() {
  if (
    confirm(
      `Execute ${dedupeStore.mode.toUpperCase()} deduplication on ${dedupeStore.remote}:${dedupeStore.path || '/'}? This may permanently remove redundant duplicate files.`
    )
  ) {
    dedupeStore.runDedupe(false)
    toast.warn(`Executing ${dedupeStore.mode.toUpperCase()} deduplication resolution`, 'Dedupe Execution')
  }
}

function openDaemonLogs() {
  if (!dedupeStore.runId) return
  const mockEntry: LogEntry = {
    time: '--:--:--',
    level: 'INFO   ',
    levelClass: 'text-slate-400',
    msg: `Deduplication run ${dedupeStore.runId}`,
    run_id: dedupeStore.runId,
    type: 'dedupe',
  }
  logsStore.openDaemonModal(mockEntry)
}

async function handlePreview(item: BrowseItem) {
  showPreviewModal.value = true
  previewLoading.value = true
  previewError.value = null
  previewData.value = null

  try {
    const filePath = item.Path || item.Name || ''
    const params = new URLSearchParams({
      remote: dedupeStore.remote,
      path: filePath,
    })
    const itemId = item.ID || item.Id
    if (itemId) {
      params.set('id', itemId)
    }
    const res = await api.get<FilePreviewResponse>(`/api/explorer/preview?${params.toString()}`)
    previewData.value = res
  } catch (err: any) {
    previewError.value = err.message || 'Failed to load file preview'
  } finally {
    previewLoading.value = false
  }
}
</script>
