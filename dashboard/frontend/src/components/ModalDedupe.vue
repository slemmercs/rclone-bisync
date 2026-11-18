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
        <div v-if="activeTab === 'items'" class="flex-1 flex flex-col min-h-0 overflow-hidden">
          <!-- Collision Search & Filter Controls Toolbar -->
          <div class="p-3 bg-carbon-surface/80 border-b border-carbon-border flex flex-col sm:flex-row sm:items-center justify-between gap-2 shrink-0">
            <!-- Search Input -->
            <div class="relative flex-1 max-w-sm">
              <i class="pi pi-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500 text-xs"></i>
              <input
                v-model="collisionSearch"
                type="text"
                placeholder="Filter by filename, directory, or ID..."
                class="w-full pl-7 pr-7 py-1 bg-carbon-canvas border border-carbon-border rounded-lg text-xs font-mono text-slate-200 placeholder-slate-500 focus:outline-none focus:border-amber-500 transition-colors"
              />
              <button
                v-if="collisionSearch"
                @click="collisionSearch = ''"
                class="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white cursor-pointer"
                title="Clear search"
              >
                <i class="pi pi-times text-[10px]"></i>
              </button>
            </div>

            <!-- Segment Filters & Actions -->
            <div class="flex items-center gap-1.5 flex-wrap font-mono text-xs">
              <button
                v-for="pill in collisionFilterOptions"
                :key="pill.value"
                @click="collisionFilter = pill.value"
                class="px-2 py-0.5 rounded text-[10px] font-semibold border transition-all cursor-pointer flex items-center gap-1 select-none"
                :class="collisionFilter === pill.value
                  ? 'bg-amber-950 text-amber-300 border-amber-600/80 shadow-sm'
                  : 'bg-carbon-canvas text-slate-400 border-carbon-border hover:bg-carbon-elevated hover:text-slate-200'"
              >
                <span>{{ pill.label }}</span>
                <span class="text-[9px] opacity-80 font-mono">({{ pill.count }})</span>
              </button>

              <div class="h-4 w-px bg-carbon-border mx-1 hidden sm:block"></div>

              <button
                type="button"
                @click="collapseAll"
                class="px-2 py-0.5 rounded bg-carbon-canvas border border-carbon-border text-slate-400 hover:text-white text-[10px] cursor-pointer"
                title="Collapse all collision items"
              >
                Collapse All
              </button>
              <button
                type="button"
                @click="expandAll"
                class="px-2 py-0.5 rounded bg-carbon-canvas border border-carbon-border text-slate-400 hover:text-white text-[10px] cursor-pointer"
                title="Expand all collision items"
              >
                Expand All
              </button>
            </div>
          </div>

          <!-- Items Scroll Area -->
          <div class="flex-1 overflow-y-auto p-4 space-y-3 font-mono">
            <div v-if="dedupeStore.parsed.items.length === 0" class="py-20 text-center text-slate-500 text-xs">
              <i class="pi pi-check-circle text-3xl text-emerald-500/50 mb-2 block"></i>
              <p>{{ dedupeStore.loading ? 'Scanning remote storage for duplicates...' : 'No duplicate files detected. All files are unique.' }}</p>
            </div>

            <!-- Empty Search Results -->
            <div v-else-if="filteredItems.length === 0" class="py-16 text-center text-slate-400 text-xs space-y-2">
              <i class="pi pi-filter-slash text-2xl text-slate-500 block"></i>
              <p>No duplicate collisions match the filter criteria.</p>
              <button
                @click="resetCollisionFilters"
                class="px-3 py-1 rounded bg-carbon-elevated border border-carbon-border text-amber-300 text-xs font-mono hover:bg-carbon-surface cursor-pointer"
              >
                Reset Filters
              </button>
            </div>

          <div
            v-for="item in filteredItems"
            :key="item.path"
            class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-2"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex items-center gap-2">
                <button
                  type="button"
                  @click="toggleCollapse(item.path)"
                  class="p-1 rounded text-slate-400 hover:text-white hover:bg-carbon-elevated transition-colors cursor-pointer shrink-0"
                  :title="collapsedItems[item.path] ? 'Expand duplicate copies' : 'Collapse duplicate copies'"
                >
                  <i :class="collapsedItems[item.path] ? 'pi pi-chevron-right' : 'pi pi-chevron-down'" class="text-xs"></i>
                </button>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-white font-bold text-xs truncate">{{ item.name }}</span>
                    <span class="text-[9px] px-1.5 py-0.2 rounded bg-amber-950 text-amber-300 border border-amber-800/60 uppercase font-bold">
                      {{ item.found }} Copies
                    </span>
                    <span v-if="item.isDir" class="text-[9px] px-1.5 py-0.2 rounded bg-indigo-950 text-indigo-300 border border-indigo-800">DIR</span>
                  </div>
                  <div class="text-[10px] text-slate-400 truncate mt-0.5">{{ item.path }}</div>
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <button
                  type="button"
                  @click="inspectInExplorer(item)"
                  class="px-2 py-0.5 rounded bg-carbon-elevated hover:bg-slate-700 text-slate-300 hover:text-white text-[10px] font-mono border border-carbon-border flex items-center gap-1 cursor-pointer transition-colors"
                  title="Inspect file path in Dual-Pane File Explorer"
                >
                  <i class="pi pi-folder-open text-[10px] text-amber-400"></i>
                  <span class="hidden sm:inline">Explorer</span>
                </button>

                <!-- Staged Decision Badge -->
                <span
                  v-if="dedupeStore.stagedActions[item.path]"
                  class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider shrink-0"
                  :class="dedupeStore.stagedActions[item.path].action === 'delete' ? 'bg-rose-950 text-rose-300 border border-rose-800' : 'bg-emerald-950 text-emerald-300 border border-emerald-800'"
                >
                  {{ dedupeStore.stagedActions[item.path].action }}
                </span>
              </div>
            </div>

            <!-- Candidates Details -->
            <div v-if="!collapsedItems[item.path] && item.candidates.length > 0" class="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-2 border-t border-carbon-border/40 text-[10px]">
              <div
                v-for="(cand, cIdx) in item.candidates"
                :key="cIdx"
                class="p-2 rounded bg-carbon-canvas border border-carbon-border flex items-center justify-between gap-2"
              >
                <div class="truncate min-w-0">
                  <span class="text-slate-400 mr-1.5">#{{ cand.index || cIdx + 1 }}</span>
                  <span v-if="cand.size !== null" class="text-slate-200 mr-2">{{ formatBytes(cand.size) }}</span>
                  <span v-if="cand.modtime" class="text-slate-400">{{ cand.modtime }}</span>
                  <div v-if="cand.id" class="text-[9px] text-slate-500 font-mono truncate">ID: {{ cand.id }}</div>
                  <div v-if="getStagedCand(item, cand)?.new_name" class="text-[9px] text-amber-300 font-mono truncate">
                    ➔ {{ getStagedCand(item, cand)?.new_name }}
                  </div>
                </div>

                <div class="flex items-center gap-1 shrink-0">
                  <template v-if="!getStagedCand(item, cand)">
                    <button
                      type="button"
                      @click="dedupeStore.stageKeepOnly(item, cand)"
                      class="px-1.5 py-0.5 rounded bg-emerald-950 hover:bg-emerald-900 text-emerald-300 border border-emerald-800/60 text-[9px] cursor-pointer"
                      title="Keep this copy and stage delete on duplicates"
                    >Keep</button>
                    <button
                      type="button"
                      @click="promptRename(item, cand)"
                      class="px-1.5 py-0.5 rounded bg-amber-950 hover:bg-amber-900 text-amber-300 border border-amber-800/60 text-[9px] cursor-pointer"
                      title="Rename this copy"
                    >Rename</button>
                    <button
                      type="button"
                      @click="dedupeStore.stageCandidateAction(item.path, cand, 'delete')"
                      class="px-1.5 py-0.5 rounded bg-rose-950 hover:bg-rose-900 text-rose-300 border border-rose-800/60 text-[9px] cursor-pointer"
                      title="Delete this copy"
                    >Delete</button>
                  </template>
                  <template v-else>
                    <span
                      class="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold uppercase border"
                      :class="{
                        'bg-emerald-950/90 text-emerald-300 border-emerald-600/60': getStagedCand(item, cand)?.action === 'keep',
                        'bg-rose-950/90 text-rose-300 border-rose-600/60': getStagedCand(item, cand)?.action === 'delete',
                        'bg-amber-950/90 text-amber-300 border-amber-600/60': getStagedCand(item, cand)?.action === 'rename',
                      }"
                    >
                      {{ getStagedCand(item, cand)?.action }}
                    </span>
                    <button
                      type="button"
                      @click="unstageCand(item, cand)"
                      class="p-0.5 text-slate-400 hover:text-rose-300 cursor-pointer"
                      title="Clear staged action"
                    >
                      <i class="pi pi-times text-[10px]"></i>
                    </button>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>
        </div>

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
        <div
          v-if="dedupeStore.stagedCount > 0"
          class="p-3 bg-carbon-surface border-t border-carbon-border flex flex-wrap items-center justify-between gap-3 shrink-0"
        >
          <div class="flex items-center gap-3">
            <button
              type="button"
              @click="dedupeStore.toggleStagedDrawer()"
              class="px-2.5 py-1 rounded-lg bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border border-amber-500/40 font-mono text-xs font-bold flex items-center gap-1.5 transition-colors cursor-pointer"
              title="Toggle Staged Actions Drawer"
            >
              <span>{{ dedupeStore.stagedCount }} staged</span>
              <i :class="dedupeStore.drawerOpen ? 'pi pi-chevron-down' : 'pi pi-chevron-up'" class="text-[10px]"></i>
            </button>
            <label class="flex items-center gap-1.5 text-[11px] font-mono text-slate-400 cursor-pointer select-none">
              <input
                type="checkbox"
                v-model="dedupeStore.triggerResync"
                class="rounded bg-carbon-canvas border-carbon-border text-amber-500 focus:ring-0 w-3.5 h-3.5"
              />
              <span>Trigger profile --resync</span>
            </label>
          </div>

          <div class="flex items-center gap-2">
            <button
              type="button"
              @click="dedupeStore.clearStagedActions()"
              class="px-3 py-1 rounded-lg bg-carbon-elevated hover:bg-slate-700 text-slate-300 text-xs font-mono border border-carbon-border transition-colors cursor-pointer"
            >
              Clear
            </button>
            <button
              type="button"
              @click="handleApplyBatch()"
              :disabled="dedupeStore.loading"
              class="px-3.5 py-1 rounded-lg bg-amber-500 hover:bg-amber-400 text-black font-bold text-xs font-mono shadow transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
            >
              <i class="pi pi-check"></i>
              <span>Apply Staged Batch</span>
            </button>
          </div>
        </div>
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
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import { api } from '../api/client'
import { useDedupeStore } from '../stores/dedupe'
import { useLogsStore } from '../stores/logs'
import { useExplorerStore } from '../stores/explorer'
import { useAppStore } from '../stores/app'
import { formatBytes } from '../composables/useFormatters'
import { useToastNotification } from '../composables/useToastNotification'
import DedupeStagedDrawer from './dedupe/DedupeStagedDrawer.vue'
import DedupeExplorerTab from './dedupe/DedupeExplorerTab.vue'
import ModalFilePreview from './ModalFilePreview.vue'
import type { LogEntry } from '../types/logs'
import type { BrowseItem } from '../types/models'
import type { FilePreviewResponse } from '../types/explorer'
import type { ParsedDedupeItem, ParsedCandidate } from '../utils/dedupeParser'
import type { DedupeStageAction } from '../types/dedupe'

const dedupeStore = useDedupeStore()
const logsStore = useLogsStore()
const explorerStore = useExplorerStore()
const appStore = useAppStore()
const toast = useToastNotification()
const activeTab = ref<'items' | 'explorer' | 'raw'>('items')

// Candidate search, filter, and collapse state
const collisionSearch = ref<string>('')
type CollisionFilterType = 'all' | 'staged' | 'unresolved' | 'multi'
const collisionFilter = ref<CollisionFilterType>('all')
const collapsedItems = ref<Record<string, boolean>>({})

function toggleCollapse(path: string) {
  collapsedItems.value[path] = !collapsedItems.value[path]
}

function expandAll() {
  collapsedItems.value = {}
}

function collapseAll() {
  const map: Record<string, boolean> = {}
  for (const item of dedupeStore.parsed.items) {
    map[item.path] = true
  }
  collapsedItems.value = map
}

function resetCollisionFilters() {
  collisionSearch.value = ''
  collisionFilter.value = 'all'
}

function inspectInExplorer(item: ParsedDedupeItem) {
  explorerStore.jumpToPath(item.path)
  appStore.switchTab('explorer')
  dedupeStore.closeModal()
}

const collisionFilterOptions = computed(() => {
  const items = dedupeStore.parsed.items || []
  const all = items.length
  let staged = 0
  let unresolved = 0
  let multi = 0

  for (const item of items) {
    const hasStaged = dedupeStore.stagedActions[item.path] || item.candidates.some(cand => !!getStagedCand(item, cand))
    if (hasStaged) {
      staged++
    } else {
      unresolved++
    }
    if ((item.found || item.candidates.length) > 2) {
      multi++
    }
  }

  return [
    { label: 'All', value: 'all' as CollisionFilterType, count: all },
    { label: 'Unresolved', value: 'unresolved' as CollisionFilterType, count: unresolved },
    { label: 'Staged', value: 'staged' as CollisionFilterType, count: staged },
    { label: 'Multi-Copy (>2)', value: 'multi' as CollisionFilterType, count: multi },
  ]
})

const filteredItems = computed(() => {
  let list = dedupeStore.parsed.items || []

  if (collisionFilter.value === 'staged') {
    list = list.filter(item => {
      if (dedupeStore.stagedActions[item.path]) return true
      return item.candidates.some(cand => !!getStagedCand(item, cand))
    })
  } else if (collisionFilter.value === 'unresolved') {
    list = list.filter(item => {
      if (dedupeStore.stagedActions[item.path]) return false
      return !item.candidates.some(cand => !!getStagedCand(item, cand))
    })
  } else if (collisionFilter.value === 'multi') {
    list = list.filter(item => (item.found || item.candidates.length) > 2)
  }

  const q = collisionSearch.value.trim().toLowerCase()
  if (q) {
    list = list.filter(item => {
      if (item.name.toLowerCase().includes(q) || item.path.toLowerCase().includes(q)) return true
      return item.candidates.some(cand => 
        (cand.id && cand.id.toLowerCase().includes(q)) ||
        (cand.raw && cand.raw.toLowerCase().includes(q))
      )
    })
  }

  return list
})

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

function getStagedCand(collision: ParsedDedupeItem, cand: ParsedCandidate): DedupeStageAction | null {
  return dedupeStore.getStagedCandidateAction(collision.path, cand)
}

function unstageCand(collision: ParsedDedupeItem, cand: ParsedCandidate) {
  const key = dedupeStore.getCandidateKey(collision.path, cand.id, cand.index)
  dedupeStore.unstageCandidateAction(key)
}

function promptRename(collision: ParsedDedupeItem, cand: ParsedCandidate) {
  const orig = collision.name || 'file'
  const lastDot = orig.lastIndexOf('.')
  const ext = lastDot !== -1 ? orig.substring(lastDot) : ''
  const base = lastDot !== -1 ? orig.substring(0, lastDot) : orig
  const defaultNew = `${base}_dedup_${cand.index || 1}${ext}`
  const newName = window.prompt(`Enter new unique name for duplicate copy #${cand.index || 1}:`, defaultNew)
  if (newName && newName.trim() && newName.trim() !== orig) {
    dedupeStore.stageCandidateAction(collision.path, cand, 'rename', newName.trim())
    toast.info(`Staged rename to ${newName.trim()}`, 'Dedupe Rename')
  }
}

async function handleApplyBatch() {
  const res = await dedupeStore.applyBatch()
  if (res && res.status !== 'error') {
    toast.success(`Batch deduplication executed successfully (${res.success_count} actions applied).`, 'Dedupe Batch')
  } else if (dedupeStore.error) {
    toast.error(dedupeStore.error, 'Dedupe Batch Failed')
  }
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
