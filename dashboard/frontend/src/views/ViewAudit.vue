<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAuditStore } from '../stores/audit'
import { useProfilesStore } from '../stores/profiles'
import ModalAuditReset from '../components/ModalAuditReset.vue'
import ModalAuditDetails from '../components/ModalAuditDetails.vue'
import type { AuditRecord } from '../types/models'

const auditStore = useAuditStore()
const profilesStore = useProfilesStore()

onMounted(async () => {
  await Promise.all([
    auditStore.fetchAuditRecords(),
    auditStore.verifyAuditLedger(),
    profilesStore.fetchProfiles(),
  ])
})

const filteredRecords = computed<AuditRecord[]>(() => {
  let list = auditStore.records
  if (auditStore.searchQuery) {
    const q = auditStore.searchQuery.toLowerCase().trim()
    list = list.filter((r) => {
      const matchAction = (r.action || '').toLowerCase().includes(q)
      const matchActor = (r.actor || '').toLowerCase().includes(q)
      const matchHash = (r.hash || '').toLowerCase().includes(q)
      const matchProfile = (r.profile_id || '').toLowerCase().includes(q) || (r.profile_name || '').toLowerCase().includes(q)
      const matchDetails = JSON.stringify(r.details || {}).toLowerCase().includes(q)
      return matchAction || matchActor || matchHash || matchProfile || matchDetails
    })
  }
  return list
})

function formatTimestamp(ts?: string): string {
  if (!ts) return '--'
  try {
    const d = new Date(ts)
    if (isNaN(d.getTime())) return ts
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
  } catch {
    return ts
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
  <div class="space-y-6">
    <!-- Cryptographic Ledger Continuity Banner -->
    <div
      class="rounded-2xl p-5 border transition-all"
      :class="
        auditStore.verification.valid
          ? 'bg-carbon-surface border-emerald-500/30 shadow-lg shadow-emerald-950/20'
          : 'bg-rose-950/40 border-rose-500/50 shadow-lg shadow-rose-950/30'
      "
    >
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div class="flex items-start sm:items-center gap-3.5">
          <div
            class="p-2.5 rounded-xl shrink-0"
            :class="
              auditStore.verification.valid
                ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
            "
          >
            <i :class="auditStore.verification.valid ? 'pi pi-shield' : 'pi pi-exclamation-triangle'" class="text-xl"></i>
          </div>
          <div>
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="text-base font-bold text-white tracking-tight flex items-center gap-2">
                <span>{{ auditStore.verification.valid ? 'Cryptographically Verified' : 'Cryptographic Integrity Fault' }}</span>
                <span
                  class="text-[10px] font-mono font-semibold px-2 py-0.5 rounded-full"
                  :class="
                    auditStore.verification.valid
                      ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
                      : 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                  "
                >
                  {{ auditStore.verification.total_records }} blocks • SHA-256 chained
                </span>
              </h2>
            </div>
            <p class="text-xs text-slate-400 mt-1 flex flex-wrap items-center gap-x-3 gap-y-1">
              <span v-if="auditStore.verification.valid">Immutable append-only ledger continuity confirmed from Genesis block.</span>
              <span v-else class="text-rose-300 font-semibold">{{ auditStore.verification.error || 'Chain continuity failure.' }}</span>
              <span class="text-slate-500">•</span>
              <span class="text-slate-500 font-mono text-[11px]">Verified: {{ auditStore.verification.checked_at || '--' }}</span>
            </p>
          </div>
        </div>

        <!-- Actions & Export Buttons -->
        <div class="flex flex-wrap items-center gap-2 self-start lg:self-auto">
          <div v-if="auditStore.verification.head_hash" class="bg-carbon-bg border border-carbon-border px-3 py-1.5 rounded-xl font-mono text-[11px] text-slate-300 flex items-center gap-2">
            <span class="text-slate-500">HEAD:</span>
            <span class="text-electric-blue">{{ auditStore.verification.head_hash.substring(0, 10) }}...</span>
            <button
              type="button"
              @click="auditStore.copyHeadHash(auditStore.verification.head_hash)"
              class="text-slate-400 hover:text-white transition-colors flex items-center justify-center p-0.5 rounded hover:bg-carbon-surface cursor-pointer"
              title="Copy head hash"
            >
              <i :class="auditStore.copiedHash ? 'pi pi-check text-emerald-400' : 'pi pi-copy'" class="text-xs"></i>
            </button>
          </div>

          <button
            type="button"
            @click="auditStore.verifyAuditLedger()"
            :disabled="auditStore.verification.checking"
            class="px-3.5 py-1.5 bg-electric-blue hover:bg-sky-400 disabled:opacity-50 text-black rounded-xl text-xs font-bold font-mono transition-all flex items-center gap-1.5 cursor-pointer shadow"
          >
            <i v-if="auditStore.verification.checking" class="pi pi-spin pi-spinner text-xs"></i>
            <span>{{ auditStore.verification.checking ? 'Verifying...' : 'Verify Now' }}</span>
          </button>

          <button
            type="button"
            @click="auditStore.resetModalOpen = true"
            class="px-3 py-1.5 bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 hover:text-white rounded-xl text-xs font-medium border border-rose-800/60 transition-colors flex items-center gap-1.5 cursor-pointer"
          >
            <span>Reset Ledger</span>
          </button>

          <a href="/api/audit/export?format=jsonl" download class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-slate-200 rounded-xl text-xs font-medium border border-carbon-border transition-colors">JSONL</a>
          <a href="/api/audit/export?format=csv" download class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-slate-200 rounded-xl text-xs font-medium border border-carbon-border transition-colors">CSV</a>
        </div>
      </div>
    </div>

    <!-- Filters Strip -->
    <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-2.5 flex-1 min-w-[280px]">
        <select
          v-model="auditStore.filterProfile"
          @change="auditStore.applyFilters()"
          class="bg-carbon-bg border border-carbon-border focus:border-electric-blue rounded-lg px-3 py-1.5 text-xs text-slate-300 outline-none flex-1 min-w-[150px] font-mono"
        >
          <option value="">All Profiles</option>
          <option v-for="p in profilesStore.profiles" :key="p.id" :value="p.id">
            {{ p.name }}
          </option>
        </select>

        <select
          v-model="auditStore.filterAction"
          @change="auditStore.applyFilters()"
          class="bg-carbon-bg border border-carbon-border focus:border-electric-blue rounded-lg px-3 py-1.5 text-xs text-slate-300 outline-none font-mono"
        >
          <option value="">All Actions</option>
          <option value="SYNC_EXECUTED">SYNC_EXECUTED</option>
          <option value="DEDUPE_EXECUTED">DEDUPE_EXECUTED</option>
          <option value="DEDUPE_BATCH_EXECUTED">DEDUPE_BATCH_EXECUTED</option>
          <option value="CONFLICT_RESOLVED">CONFLICT_RESOLVED</option>
          <option value="DELETION_ACCEPTED">DELETION_ACCEPTED</option>
          <option value="DELETION_RESTORED">DELETION_RESTORED</option>
          <option value="REVIEW_SESSION_DISMISSED">REVIEW_SESSION_DISMISSED</option>
          <option value="ROLLBACK_EXECUTED">ROLLBACK_EXECUTED</option>
          <option value="SECRET_ROTATED">SECRET_ROTATED</option>
          <option value="CONFIG_MUTATED">CONFIG_MUTATED</option>
          <option value="WORKSPACE_FILE_UPLOADED">WORKSPACE_FILE_UPLOADED</option>
          <option value="WORKSPACE_ITEM_COPIED">WORKSPACE_ITEM_COPIED</option>
          <option value="WORKSPACE_ITEM_MOVED">WORKSPACE_ITEM_MOVED</option>
          <option value="WORKSPACE_ITEM_DELETED">WORKSPACE_ITEM_DELETED</option>
          <option value="WORKSPACE_DIR_PURGED">WORKSPACE_DIR_PURGED</option>
          <option value="WORKSPACE_DIR_CREATED">WORKSPACE_DIR_CREATED</option>
        </select>

        <select
          v-model="auditStore.filterActor"
          @change="auditStore.applyFilters()"
          class="bg-carbon-bg border border-carbon-border focus:border-electric-blue rounded-lg px-3 py-1.5 text-xs text-slate-300 outline-none font-mono"
        >
          <option value="">All Actors</option>
          <option value="SYSTEM">SYSTEM</option>
          <option value="USER">USER</option>
          <option value="ADMIN">ADMIN</option>
        </select>

        <div class="relative w-44">
          <input
            v-model="auditStore.searchQuery"
            type="text"
            placeholder="Search payload..."
            class="w-full bg-carbon-bg border border-carbon-border rounded-lg pl-7 pr-2.5 py-1.5 text-xs text-slate-200 placeholder-slate-500 font-mono focus:outline-none focus:border-electric-blue"
          />
          <i class="pi pi-search text-slate-500 absolute left-2.5 top-2.5 text-[10px] pointer-events-none"></i>
        </div>

        <button
          type="button"
          @click="auditStore.applyFilters()"
          class="px-3 py-1.5 bg-electric-blue hover:bg-sky-400 text-black rounded-lg text-xs font-semibold transition-colors cursor-pointer"
        >
          Filter
        </button>
        <button
          type="button"
          @click="auditStore.resetFilters()"
          class="px-2.5 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-slate-300 rounded-lg text-xs transition-colors cursor-pointer"
        >
          Reset
        </button>
      </div>

      <div class="flex items-center gap-2 text-xs text-slate-400 font-mono">
        <span>{{ auditStore.total }} total recorded events</span>
      </div>
    </div>
    <!-- Ledger Record Table -->
    <div class="bg-carbon-surface border border-carbon-border rounded-2xl overflow-hidden shadow-sm">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs font-mono">
          <thead class="bg-carbon-bg text-slate-400 uppercase tracking-wider text-[10px] font-semibold border-b border-carbon-border select-none">
            <tr>
              <th class="py-3 px-4"># Seq</th>
              <th class="py-3 px-4">Timestamp</th>
              <th class="py-3 px-4">Profile</th>
              <th class="py-3 px-4">Action</th>
              <th class="py-3 px-4">Actor</th>
              <th class="py-3 px-4">Hash Chaining</th>
              <th class="py-3 px-4 text-right">Details</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-carbon-border text-slate-300">
            <tr v-for="rec in filteredRecords" :key="rec.seq_id" class="hover:bg-carbon-elevated/40 transition-colors">
              <td class="py-3 px-4 text-electric-blue font-bold">#{{ rec.seq_id }}</td>
              <td class="py-3 px-4 text-slate-400 whitespace-nowrap">{{ formatTimestamp(rec.timestamp) }}</td>
              <td class="py-3 px-4 font-sans font-medium text-slate-200">
                <div class="flex flex-col">
                  <span>{{ rec.profile_name || rec.profile_id || '--' }}</span>
                  <span v-if="rec.profile_name && rec.profile_name !== rec.profile_id" class="text-[10px] text-slate-500 font-mono">
                    {{ rec.profile_id }}
                  </span>
                </div>
              </td>
              <td class="py-3 px-4 whitespace-nowrap">
                <span
                  class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border"
                  :class="getActionBadgeClass(rec.action)"
                >
                  {{ rec.action }}
                </span>
              </td>
              <td class="py-3 px-4 whitespace-nowrap">
                <span class="px-1.5 py-0.5 rounded text-[10px] bg-carbon-elevated text-slate-300 border border-carbon-border uppercase">
                  {{ rec.actor }}
                </span>
              </td>
              <td class="py-3 px-4 text-[11px] text-slate-400 whitespace-nowrap">
                <div class="flex items-center gap-1.5" :title="`Current Hash: ${rec.hash}\nPrevious Hash: ${rec.prev_hash}`">
                  <span class="text-electric-blue">{{ rec.hash ? rec.hash.substring(0, 10) + '...' : '--' }}</span>
                  <span class="text-slate-600">←</span>
                  <span class="text-slate-500">{{ rec.prev_hash ? rec.prev_hash.substring(0, 6) + '...' : '--' }}</span>
                </div>
              </td>
              <td class="py-3 px-4 text-right whitespace-nowrap font-sans">
                <button
                  type="button"
                  @click="auditStore.inspectRecord(rec)"
                  class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-200 hover:text-white rounded text-[11px] font-medium transition-colors border border-carbon-border shadow-sm cursor-pointer"
                >
                  Inspect
                </button>
              </td>
            </tr>

            <tr v-if="auditStore.loading">
              <td colspan="7" class="py-12 text-center text-slate-500 animate-pulse font-sans">
                Loading audit ledger records...
              </td>
            </tr>

            <tr v-else-if="filteredRecords.length === 0">
              <td colspan="7" class="py-12 text-center text-slate-500 font-sans">
                No audit events match the current filter criteria.
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination Footer -->
      <div class="bg-carbon-bg px-4 py-3 border-t border-carbon-border flex flex-wrap items-center justify-between gap-3 text-xs font-mono">
        <div class="text-slate-400">
          <span>Page {{ auditStore.page }} of {{ auditStore.totalPages }}</span>
          <span class="text-slate-600 mx-1.5">•</span>
          <span>Showing {{ filteredRecords.length }} records</span>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            @click="auditStore.changePage(auditStore.page - 1)"
            :disabled="auditStore.page <= 1"
            class="px-3 py-1 bg-carbon-elevated hover:bg-slate-800 disabled:opacity-40 text-slate-300 rounded border border-carbon-border transition-colors cursor-pointer"
          >
            Previous
          </button>
          <button
            type="button"
            @click="auditStore.changePage(auditStore.page + 1)"
            :disabled="auditStore.page >= auditStore.totalPages"
            class="px-3 py-1 bg-carbon-elevated hover:bg-slate-800 disabled:opacity-40 text-slate-300 rounded border border-carbon-border transition-colors cursor-pointer"
          >
            Next
          </button>
        </div>
      </div>
    </div>

    <!-- Modals -->
    <ModalAuditReset />
    <ModalAuditDetails />
  </div>
</template>
