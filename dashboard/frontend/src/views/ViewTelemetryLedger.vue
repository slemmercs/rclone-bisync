<template>
  <div class="bg-carbon-surface border border-carbon-border rounded-2xl overflow-hidden flex flex-col shadow-xl">
    <!-- Header & Filter Bar -->
    <div class="p-3.5 border-b border-carbon-border flex flex-wrap items-center justify-between gap-3 bg-carbon-surface/90">
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-2">
          <i class="pi pi-list text-slate-400 text-sm"></i>
          <span class="text-xs font-mono font-semibold text-slate-200 tracking-wider">Unified Log Ledger</span>
          <span class="text-[9px] font-mono text-slate-500 bg-carbon-canvas px-1.5 py-0.5 rounded border border-carbon-border">
            bisync.log • Newest first
          </span>
        </div>

        <!-- Category Filter Pills -->
        <div class="flex items-center bg-carbon-canvas p-0.5 rounded-lg border border-carbon-border text-[10px] font-mono ml-2">
          <button
            @click="setCategoryFilter('all')"
            :class="logsStore.ledgerCategoryFilter === 'all' ? 'bg-electric-blue text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >
            All
          </button>
          <button
            @click="setCategoryFilter('file_ops')"
            :class="logsStore.ledgerCategoryFilter === 'file_ops' ? 'bg-amber-600 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
            title="Transfers, deletes, uploads, conflicts"
          >
            File Ops
          </button>
          <button
            @click="setCategoryFilter('sync')"
            :class="logsStore.ledgerCategoryFilter === 'sync' ? 'bg-indigo-600 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >
            Sync
          </button>
          <button
            @click="setCategoryFilter('audit')"
            :class="logsStore.ledgerCategoryFilter === 'audit' ? 'bg-purple-600 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >
            Audit
          </button>
        </div>

        <!-- Profile Dropdown Filter -->
        <div class="flex items-center ml-1">
          <select
            v-model="logsStore.ledgerProfileFilter"
            @change="reloadLogs(1)"
            class="bg-carbon-canvas border border-carbon-border text-slate-300 text-[10px] font-mono rounded px-2 py-0.5 focus:ring-0 focus:border-electric-blue cursor-pointer"
          >
            <option value="all">All Profiles</option>
            <option v-for="p in profilesStore.profiles" :key="p.id" :value="p.id">
              {{ p.name || p.id }}
            </option>
          </select>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <!-- File Transfers Suppression Toggle -->
        <label class="flex items-center gap-1.5 text-[11px] font-mono text-slate-400 cursor-pointer select-none" title="Show individual per-file transfer records (sync_file)">
          <input
            type="checkbox"
            v-model="logsStore.includeFileTransfers"
            @change="reloadLogs(1)"
            class="rounded bg-carbon-canvas border-carbon-border text-electric-blue focus:ring-0 w-3.5 h-3.5"
          >
          <span>File Transfers</span>
        </label>

        <!-- Inline Log Folding Controls -->
        <label class="flex items-center gap-1.5 text-[11px] font-mono text-slate-400 cursor-pointer select-none">
          <input
            type="checkbox"
            v-model="logsStore.foldRepetitive"
            class="rounded bg-carbon-canvas border-carbon-border text-electric-blue focus:ring-0 w-3.5 h-3.5"
          >
          <span>Fold Repeated</span>
        </label>
        <button
          v-if="logsStore.foldRepetitive"
          @click="logsStore.toggleAllClusters()"
          class="text-[10px] font-mono px-2 py-0.5 rounded bg-carbon-elevated hover:bg-slate-700 text-slate-300 border border-carbon-border transition-colors cursor-pointer"
        >
          {{ logsStore.areAllClustersExpanded ? 'Collapse All' : 'Expand All' }}
        </button>

        <!-- Daemon Stream Modal Button -->
        <button
          @click="logsStore.openDaemonModal(null)"
          class="text-[11px] font-mono px-2 py-0.5 rounded bg-electric-blue/10 hover:bg-electric-blue/20 text-electric-blue border border-electric-blue/30 transition-colors flex items-center gap-1 cursor-pointer"
          title="Open correlated daemon log viewer"
        >
          <i class="pi pi-terminal text-xs"></i>
          <span>Daemon Stream</span>
        </button>

        <a
          href="/api/logs/bisync/raw"
          target="_blank"
          rel="noopener noreferrer"
          class="text-[11px] font-mono text-slate-400 hover:text-slate-200 transition-colors flex items-center gap-1"
        >
          <span>Raw</span>
          <i class="pi pi-external-link text-xs"></i>
        </a>
      </div>
    </div>

    <!-- Ledger Entries Table/List -->

    <div class="overflow-x-auto divide-y divide-carbon-border/60 bg-carbon-canvas min-h-[16rem] max-h-[36rem] overflow-y-auto">
      <div v-if="logsStore.loading" class="text-electric-blue animate-pulse text-center py-12 font-mono text-xs">
        <i class="pi pi-spin pi-spinner mr-2"></i>Loading unified timeline ledger...
      </div>

      <div v-else-if="logsStore.error" class="p-4 text-xs font-mono text-rose-300 bg-rose-500/10 border-b border-rose-500/20 flex items-center gap-2">
        <i class="pi pi-exclamation-triangle text-rose-400"></i>
        <span>{{ logsStore.error }}</span>
      </div>

      <div v-else-if="logsStore.clusteredLedgerLogs.length === 0" class="text-slate-500 text-center py-16 font-mono text-xs">
        No log entries matching the selected criteria.
      </div>

      <template v-else v-for="cluster in logsStore.clusteredLedgerLogs" :key="cluster.id">
        <div class="hover:bg-carbon-surface/60 transition-colors">
          <!-- Main Row -->
          <div
            @click="cluster.isCluster || cluster.isConsolidatedRun ? logsStore.toggleCluster(cluster.id) : null"
            :class="(cluster.isCluster || cluster.isConsolidatedRun) ? 'cursor-pointer' : ''"
            class="px-3 py-2 flex items-center justify-between gap-2 text-xs font-mono select-none"
          >
            <div class="flex items-center gap-2 min-w-0 flex-1">
              <!-- Expand/Collapse Chevron -->
              <span v-if="cluster.isCluster || cluster.isConsolidatedRun" class="text-slate-500 hover:text-slate-300 w-4 text-center shrink-0">
                <i class="pi text-[10px]" :class="cluster.expanded ? 'pi-chevron-down' : 'pi-chevron-right'"></i>
              </span>
              <span v-else class="w-4 shrink-0"></span>

              <!-- Timestamp -->
              <span class="text-slate-500 text-[10px] shrink-0 tabular-nums">
                {{ cluster.entry.time || '--:--:--' }}
              </span>

              <!-- Level Badge -->
              <span :class="cluster.entry.levelClass || 'text-slate-400'" class="shrink-0 text-[10px] uppercase font-bold">
                [{{ (cluster.entry.level || 'INFO').trim() }}]
              </span>

              <!-- Profile Badge -->
              <button
                v-if="cluster.entry.profile_name || (cluster.entry.profile_id && cluster.entry.profile_id !== 'default')"
                type="button"
                @click.stop="filterByProfile(cluster.entry.profile_id || cluster.entry.profile_name!)"
                class="px-1.5 py-0.5 rounded bg-indigo-950/80 text-indigo-300 hover:text-white hover:border-indigo-600 font-mono text-[9px] border border-indigo-800/50 shrink-0 transition-colors cursor-pointer"
                :title="'Filter by profile: ' + (cluster.entry.profile_name || cluster.entry.profile_id)"
              >
                [{{ cluster.entry.profile_name || cluster.entry.profile_id }}]
              </button>

              <!-- Run ID Badge (opens daemon modal) -->
              <button
                v-if="cluster.entry.run_id"
                type="button"
                @click.stop="logsStore.openDaemonModal(cluster.entry)"
                class="px-1.5 py-0.5 rounded bg-carbon-elevated text-slate-300 hover:text-electric-blue hover:border-electric-blue/50 font-mono text-[9px] border border-carbon-border shrink-0 transition-colors cursor-pointer"
                :title="'View daemon logs for run: ' + cluster.entry.run_id"
              >
                run:{{ cluster.entry.run_id.length > 8 ? cluster.entry.run_id.substring(0, 8) : cluster.entry.run_id }}
              </button>

              <!-- Audit Action Badge -->
              <span
                v-if="cluster.entry.action"
                class="px-1.5 py-0.5 rounded font-mono text-[9px] border uppercase tracking-wider shrink-0"
                :class="getAuditActionBadgeClass(cluster.entry.action)"
              >
                {{ cluster.entry.action }}
              </span>

              <!-- Message Text -->
              <span class="text-slate-300 truncate" :title="cluster.entry.msg">
                {{ cluster.entry.msg }}
              </span>
            </div>

            <!-- Cluster / Consolidated Run Counts -->
            <div class="flex items-center gap-2 shrink-0">
              <span
                v-if="cluster.isConsolidatedRun"
                class="text-[9px] px-2 py-0.5 rounded bg-indigo-950/60 text-indigo-300 border border-indigo-800/40"
              >
                {{ cluster.count }} events
              </span>
              <span
                v-else-if="cluster.isCluster"
                class="text-[9px] px-2 py-0.5 rounded bg-electric-blue/20 text-electric-blue border border-electric-blue/30 font-bold"
              >
                {{ cluster.count }}× repeated
              </span>
            </div>
          </div>

          <!-- Expanded Sub-Entries Drawer -->
          <div v-if="cluster.expanded && cluster.entries.length > 1" class="bg-carbon-surface/40 border-t border-carbon-border/40 px-6 py-2 space-y-1">
            <div
              v-for="(sub, sIdx) in cluster.entries"
              :key="sIdx"
              class="flex items-center gap-2 text-[11px] font-mono text-slate-400 py-0.5"
            >
              <span class="text-slate-600 text-[10px] tabular-nums shrink-0">{{ sub.time || '--:--:--' }}</span>
              <span :class="sub.levelClass || 'text-slate-500'" class="text-[9px] uppercase shrink-0">
                [{{ (sub.level || 'INFO').trim() }}]
              </span>
              <span class="text-slate-300 truncate flex-1">{{ sub.msg }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- Pagination Footer -->
    <div class="p-3 border-t border-carbon-border flex items-center justify-between bg-carbon-surface/90 text-xs font-mono">
      <div class="text-slate-500">
        Page <span class="text-slate-200 font-semibold">{{ logsStore.ledgerPage }}</span> of <span class="text-slate-200 font-semibold">{{ logsStore.ledgerTotalPages }}</span>
        <span class="text-slate-600 ml-2">({{ logsStore.ledgerTotal }} total entries)</span>
      </div>

      <div class="flex items-center gap-2">
        <button
          @click="changePage(logsStore.ledgerPage - 1)"
          :disabled="logsStore.ledgerPage <= 1 || logsStore.loading"
          class="px-3 py-1 bg-carbon-elevated hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed text-slate-300 rounded border border-carbon-border transition-colors cursor-pointer"
        >
          Previous
        </button>
        <button
          @click="changePage(logsStore.ledgerPage + 1)"
          :disabled="logsStore.ledgerPage >= logsStore.ledgerTotalPages || logsStore.loading"
          class="px-3 py-1 bg-carbon-elevated hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed text-slate-300 rounded border border-carbon-border transition-colors cursor-pointer"
        >
          Next
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useLogsStore } from '../stores/logs'
import { useProfilesStore } from '../stores/profiles'

const logsStore = useLogsStore()
const profilesStore = useProfilesStore()

function getAuditActionBadgeClass(action?: string) {
  if (!action) return 'bg-slate-800 text-slate-300 border-slate-700'
  const act = action.toUpperCase()
  if (act.includes('DELETE') || act.includes('PURGE')) return 'bg-rose-950/80 text-rose-300 border-rose-800/60'
  if (act.includes('CREATE') || act.includes('MKDIR') || act.includes('RESTORE')) return 'bg-emerald-950/80 text-emerald-300 border-emerald-800/60'
  if (act.includes('SYNC') || act.includes('TRANSFER')) return 'bg-indigo-950/80 text-indigo-300 border-indigo-800/60'
  if (act.includes('DEDUPE')) return 'bg-amber-950/80 text-amber-300 border-amber-800/60'
  return 'bg-purple-950/80 text-purple-300 border-purple-800/60'
}

function setCategoryFilter(filter: 'all' | 'file_ops' | 'sync' | 'audit') {
  logsStore.ledgerCategoryFilter = filter
  reloadLogs(1)
}

function filterByProfile(profileIdOrName: string) {
  // If matched to a profile ID or name
  const found = profilesStore.profiles.find(p => p.id === profileIdOrName || p.name === profileIdOrName)
  logsStore.ledgerProfileFilter = found ? found.id : profileIdOrName
  reloadLogs(1)
}

function changePage(newPage: number) {
  if (newPage >= 1 && newPage <= logsStore.ledgerTotalPages) {
    reloadLogs(newPage)
  }
}

function reloadLogs(page = 1) {
  logsStore.fetchBisyncLogs(page, profilesStore.profiles)
}

onMounted(() => {
  if (logsStore.rawBisyncLogs.length === 0) {
    reloadLogs(1)
  }
})
</script>

