<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRemotesStore } from '../stores/remotes'
import { useDedupeStore } from '../stores/dedupe'
import ModalDedupe from '../components/ModalDedupe.vue'
import ModalRemoteReconnect from '../components/ModalRemoteReconnect.vue'
import ModalOAuthMigration from '../components/ModalOAuthMigration.vue'
import ModalAuthAudit from '../components/ModalAuthAudit.vue'
import ModalHasherManager from '../components/ModalHasherManager.vue'
import ModalCryptWrap from '../components/ModalCryptWrap.vue'

const remotesStore = useRemotesStore()
const dedupeStore = useDedupeStore()

const categoryFilter = ref<'all' | 'cloud' | 'object' | 'protocol' | 'virtual'>('all')
const searchQuery = ref<string>('')

const remotes = computed(() => remotesStore.remotes)
const summary = computed(() => remotesStore.summary)
const loading = computed(() => remotesStore.loading)

const quotaPct = computed(() => {
  const used = Number(summary.value?.total_used_bytes || 0)
  const total = Number(summary.value?.total_capacity_bytes || 0)
  if (total <= 0) return '0.0'
  return ((used / total) * 100).toFixed(1)
})

const filteredRemotes = computed(() => {
  let list = remotes.value || []

  if (categoryFilter.value === 'cloud') {
    list = list.filter(r => ['drive', 'onedrive', 'dropbox', 'box', 'mega', 'pcloud'].includes((r.type || '').toLowerCase()))
  } else if (categoryFilter.value === 'object') {
    list = list.filter(r => ['s3', 'b2', 'gcs', 'azureblob', 'wasabi', 'minio'].includes((r.type || '').toLowerCase()))
  } else if (categoryFilter.value === 'protocol') {
    list = list.filter(r => ['sftp', 'ftp', 'webdav', 'smb', 'http'].includes((r.type || '').toLowerCase()))
  } else if (categoryFilter.value === 'virtual') {
    list = list.filter(r => ['hasher', 'crypt', 'alias', 'combine', 'chunker', 'union'].includes((r.type || '').toLowerCase()))
  }

  const q = searchQuery.value.toLowerCase().trim()
  if (q) {
    list = list.filter(r => (r.name || '').toLowerCase().includes(q) || (r.type || '').toLowerCase().includes(q))
  }

  return list
})

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '-- GB'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function getQuotaPct(used?: number, total?: number): number {
  if (!used || !total || total <= 0) return 0
  const pct = (used / total) * 100
  return Math.min(Math.max(pct, 0), 100)
}

function getAuthBadge(status?: string): { text: string; dot: string; textClass: string } {
  switch (status) {
    case 'connected':
      return { text: 'Connected', dot: 'bg-emerald-400', textClass: 'text-emerald-400' }
    case 'migration_required':
      return { text: 'Migration Needed', dot: 'bg-amber-400 animate-pulse', textClass: 'text-amber-400' }
    case 'authorization_required':
      return { text: 'Auth Required', dot: 'bg-rose-400 animate-pulse', textClass: 'text-rose-400' }
    case 'configuration_required':
      return { text: 'Config Required', dot: 'bg-sky-400', textClass: 'text-sky-400' }
    case 'error':
      return { text: 'Auth Error', dot: 'bg-rose-500', textClass: 'text-rose-500' }
    default:
      return { text: 'Status Unknown', dot: 'bg-slate-400', textClass: 'text-slate-400' }
  }
}

function checkAuth(name?: string) {
  remotesStore.checkAuth(name)
}

function confirmDelete(name: string) {
  if (confirm(`Permanently delete remote configuration "${name}"? Active sync profiles referencing this remote will be impacted.`)) {
    remotesStore.deleteRemote(name)
  }
}

onMounted(() => {
  remotesStore.fetchRemotes()
})
</script>

<template>
  <div class="space-y-6">
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Total Cloud Space</div>
        <div class="text-xl font-bold text-white mt-2 font-mono">{{ formatBytes(summary?.total_capacity_bytes) }}</div>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Utilized Storage</div>
        <div class="text-xl font-bold text-electric-blue mt-2 font-mono">{{ formatBytes(summary?.total_used_bytes) }}</div>
        <div class="text-[10px] text-slate-400 mt-1">{{ quotaPct }}% used</div>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Available Space</div>
        <div class="text-xl font-bold text-emerald-400 mt-2 font-mono">{{ formatBytes(summary?.total_free_bytes) }}</div>
      </div>
      <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border">
        <div class="text-[11px] text-slate-400 uppercase font-semibold">Remotes Count</div>
        <div class="text-xl font-bold text-white mt-2 font-mono">{{ remotes.length }}</div>
      </div>
    </div>

    <!-- Toolbar with Search and Add Remote -->
    <div class="bg-carbon-surface p-4 rounded-2xl border border-carbon-border flex flex-wrap items-center justify-between gap-4">
      <div class="flex flex-wrap items-center gap-3">
        <div class="flex items-center gap-2">
          <div class="p-2.5 rounded-xl bg-electric-blue/10 border border-electric-blue/30 text-electric-blue">
            <i class="pi pi-cloud text-base"></i>
          </div>
          <div>
            <span class="font-bold text-white text-sm">Storage Remotes</span>
            <div class="text-[11px] text-slate-400">Cloud object storage endpoints</div>
          </div>
        </div>

        <!-- Category Filter Pills -->
        <div class="flex items-center bg-carbon-canvas p-0.5 rounded-lg border border-carbon-border text-[10px] font-mono ml-2">
          <button
            type="button"
            @click="categoryFilter = 'all'"
            :class="categoryFilter === 'all' ? 'bg-electric-blue text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >All</button>
          <button
            type="button"
            @click="categoryFilter = 'cloud'"
            :class="categoryFilter === 'cloud' ? 'bg-electric-blue text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >Cloud</button>
          <button
            type="button"
            @click="categoryFilter = 'object'"
            :class="categoryFilter === 'object' ? 'bg-electric-blue text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >Object</button>
          <button
            type="button"
            @click="categoryFilter = 'protocol'"
            :class="categoryFilter === 'protocol' ? 'bg-electric-blue text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >Protocols</button>
          <button
            type="button"
            @click="categoryFilter = 'virtual'"
            :class="categoryFilter === 'virtual' ? 'bg-electric-blue text-white font-semibold' : 'text-slate-400 hover:text-slate-200'"
            class="px-2 py-0.5 rounded transition-all cursor-pointer"
          >Virtual</button>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <div class="relative w-44">
          <input
            type="text"
            v-model="searchQuery"
            placeholder="Search remotes..."
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg pl-7 pr-2.5 py-1 text-xs text-slate-200 placeholder-slate-500 font-mono focus:outline-none focus:border-electric-blue"
          />
          <i class="pi pi-search text-slate-500 absolute left-2 top-2 text-[10px] pointer-events-none"></i>
        </div>

        <button
          @click="checkAuth()"
          :disabled="loading"
          class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-electric-blue rounded-xl text-xs font-semibold border border-carbon-border transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
        >
          <i class="pi pi-refresh text-xs" :class="{ 'pi-spin': loading }"></i>
          <span>Check Auth</span>
        </button>

        <button
          @click="remotesStore.openHasherManager('', 'manager')"
          class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-amber-400 rounded-xl text-xs font-semibold border border-carbon-border transition-colors flex items-center gap-1.5 cursor-pointer"
          title="Open Hasher Acceleration & Cache Console"
        >
          <i class="pi pi-bolt text-xs text-amber-400"></i>
          <span>Hasher</span>
        </button>

        <button
          @click="remotesStore.viewAuthAudit()"
          class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-slate-300 rounded-xl text-xs font-semibold border border-carbon-border transition-colors flex items-center gap-1.5 cursor-pointer"
          title="OAuth Authentication Audit Trail"
        >
          <i class="pi pi-history text-xs text-electric-blue"></i>
          <span>Auth Audit</span>
        </button>

        <button
          @click="dedupeStore.openWizard()"
          class="px-3.5 py-1.5 bg-electric-blue hover:bg-sky-400 text-black font-bold rounded-xl text-xs font-mono transition-colors flex items-center gap-1.5 cursor-pointer shadow"
        >
          <i class="pi pi-plus text-xs"></i>
          <span>Add Remote</span>
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="filteredRemotes.length === 0" class="p-12 text-center text-slate-500 bg-carbon-surface border border-carbon-border rounded-2xl">
      <i class="pi pi-cloud text-4xl text-slate-600 mb-3 block"></i>
      <p class="text-sm font-semibold text-slate-400">No matching storage remotes found</p>
    </div>

    <!-- Remote Cards Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div
        v-for="remote in filteredRemotes"
        :key="remote.name"
        class="bg-carbon-surface border border-carbon-border hover:border-carbon-border/90 rounded-2xl p-5 space-y-3 transition-colors flex flex-col justify-between"
      >
        <div class="space-y-3">
          <div class="flex items-start justify-between">
            <div>
              <h4 class="font-bold text-white text-sm font-mono flex items-center gap-2">
                <span>{{ remote.name }}</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded uppercase bg-carbon-elevated text-slate-400 border border-carbon-border font-mono">
                  {{ remote.type || 'remote' }}
                </span>
                <span v-if="(remote.type || '').toLowerCase() === 'hasher'" class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-amber-950/60 text-amber-300 border border-amber-800/60 font-mono flex items-center gap-1">
                  <i class="pi pi-bolt text-[8px]"></i> Accelerated
                </span>
                <span v-if="(remote.type || '').toLowerCase() === 'crypt'" class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-cyan-950/60 text-cyan-300 border border-cyan-800/60 font-mono flex items-center gap-1">
                  <i class="pi pi-shield text-[8px]"></i> Encrypted
                </span>
                <span v-if="remote.is_active_target" class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-800/60 font-mono">
                  Active
                </span>
              </h4>
              <div class="flex items-center gap-2 text-[11px] text-slate-400 font-mono mt-1">
                <span>{{ remote.endpoint || remote.root || 'Configured storage backend' }}</span>
                <template v-if="remote.auth">
                  <span>•</span>
                  <span class="inline-flex items-center gap-1" :class="getAuthBadge(remote.auth.status).textClass">
                    <span class="w-1.5 h-1.5 rounded-full" :class="getAuthBadge(remote.auth.status).dot"></span>
                    <span>{{ getAuthBadge(remote.auth.status).text }}</span>
                  </span>
                </template>
              </div>
            </div>
            <span
              class="text-[10px] font-mono px-2 py-0.5 rounded-full font-bold uppercase"
              :class="remote.status === 'error' ? 'bg-rose-500/10 text-rose-400 border border-rose-500/30' : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'"
            >
              {{ remote.status === 'error' ? 'ERROR' : 'CONNECTED' }}
            </span>
          </div>

          <!-- Migration Required Warning Banner -->
          <div
            v-if="remote.auth && remote.auth.status === 'migration_required'"
            class="p-2.5 rounded-xl bg-amber-950/40 border border-amber-800/50 flex items-center justify-between gap-2 text-xs text-amber-200 font-mono"
          >
            <div class="flex items-center gap-1.5 text-[11px]">
              <i class="pi pi-exclamation-triangle text-amber-400 text-xs shrink-0"></i>
              <span>Shared Client ID. Personal credentials recommended.</span>
            </div>
            <button
              type="button"
              @click="remotesStore.openMigrationWizard(remote)"
              class="px-2 py-0.5 text-[10px] font-bold bg-amber-500 hover:bg-amber-400 text-slate-950 rounded transition-colors shrink-0 cursor-pointer"
            >
              Configure
            </button>
          </div>

          <!-- Storage Quota Section -->
          <div v-if="remote.about && (remote.about.total || remote.about.used)" class="bg-carbon-bg p-3 rounded-xl border border-carbon-border space-y-2">
            <div class="flex justify-between items-baseline text-xs font-mono">
              <span class="text-slate-400 text-[11px]">Storage Allocation</span>
              <span class="text-white font-semibold text-[11px]">
                {{ formatBytes(remote.about.used) }} / {{ formatBytes(remote.about.total) }}
              </span>
            </div>

            <!-- Multi-segment Quota Bar -->
            <div class="w-full bg-carbon-surface rounded-full h-2 overflow-hidden flex border border-carbon-border">
              <div
                class="transition-all duration-500"
                :class="getQuotaPct(remote.about.used, remote.about.total) > 90 ? 'bg-rose-500' : 'bg-electric-blue'"
                :style="`width: ${getQuotaPct(remote.about.used, remote.about.total)}%`"
              ></div>
            </div>

            <!-- Quota Breakdown Metrics -->
            <div class="grid grid-cols-3 gap-2 pt-1 text-[10px] font-mono border-t border-carbon-border/50">
              <div>
                <span class="text-slate-500 block uppercase">Used Space</span>
                <span class="text-slate-200">{{ formatBytes(remote.about.used) }}</span>
              </div>
              <div>
                <span class="text-slate-500 block uppercase">Free Space</span>
                <span class="text-emerald-400">{{ formatBytes(remote.about.free) }}</span>
              </div>
              <div>
                <span class="text-slate-500 block uppercase">Trash Bin</span>
                <span class="text-amber-400">{{ formatBytes(remote.about.trashed) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="pt-3 border-t border-carbon-border flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
          <div class="flex flex-wrap items-center gap-1.5">
            <!-- Test Connection Button -->
            <button
              type="button"
              @click="remotesStore.validateRemote(remote.name)"
              :disabled="remotesStore.validating[remote.name]"
              class="px-2.5 py-1 text-[11px] rounded-lg border transition-colors flex items-center gap-1 cursor-pointer"
              :class="
                remotesStore.validated[remote.name]
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                  : 'bg-carbon-elevated hover:bg-slate-800 text-slate-300 border-carbon-border'
              "
              title="Test connection and remote connectivity"
            >
              <i
                :class="
                  remotesStore.validating[remote.name]
                    ? 'pi pi-spin pi-spinner text-electric-blue'
                    : remotesStore.validated[remote.name]
                    ? 'pi pi-check text-emerald-400'
                    : 'pi pi-bolt text-slate-400'
                "
                class="text-xs"
              ></i>
              <span>{{ remotesStore.validated[remote.name] ? 'Connected ✓' : 'Test' }}</span>
            </button>

            <!-- OAuth Reconnect / Cancel -->
            <template v-if="remote.auth && remote.auth.is_oauth">
              <button
                v-if="!remotesStore.reconnecting[remote.name]"
                type="button"
                @click="remotesStore.reconnectRemote(remote.name)"
                class="px-2.5 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border transition-colors flex items-center gap-1 cursor-pointer"
                title="Trigger OAuth Reconnect"
              >
                <i class="pi pi-sync text-xs text-electric-blue"></i>
                <span>Reconnect</span>
              </button>
              <button
                v-else
                type="button"
                @click="remotesStore.cancelReconnect(remote.name)"
                class="px-2.5 py-1 text-[11px] rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1 cursor-pointer"
              >
                <i class="pi pi-spin pi-spinner text-xs"></i>
                <span>Cancel</span>
              </button>

              <!-- OAuth App Config -->
              <button
                type="button"
                @click="remotesStore.openMigrationWizard(remote)"
                class="px-2.5 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-slate-400 hover:text-slate-200 border border-carbon-border transition-colors flex items-center gap-1 cursor-pointer"
                title="Configure OAuth Client ID & Secret"
              >
                <i class="pi pi-key text-[10px]"></i>
                <span>OAuth</span>
              </button>
            </template>

            <!-- Hasher Remote Maintenance -->
            <button
              v-if="(remote.type || '').toLowerCase() === 'hasher'"
              type="button"
              @click="remotesStore.openHasherManager(remote.name, 'manager')"
              class="px-2.5 py-1 text-[11px] rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1 cursor-pointer"
              title="Hasher Maintenance Console"
            >
              <i class="pi pi-bolt text-xs text-amber-400"></i>
              <span>Hasher</span>
            </button>

            <!-- Virtual Overlay Actions for Standard Storage Remotes -->
            <template v-else-if="(remote.type || '').toLowerCase() !== 'crypt'">
              <button
                type="button"
                @click="remotesStore.openHasherManager(remote.name, 'wrap')"
                class="px-2 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border transition-colors flex items-center gap-1 cursor-pointer"
                title="Wrap remote with Hasher checksum acceleration overlay"
              >
                <i class="pi pi-bolt text-xs text-amber-400"></i>
                <span>Hasher</span>
              </button>

              <button
                type="button"
                @click="remotesStore.openCryptWrap(remote.name)"
                class="px-2 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border transition-colors flex items-center gap-1 cursor-pointer"
                title="Wrap remote with client-side Crypt encryption overlay"
              >
                <i class="pi pi-shield text-xs text-cyan-400"></i>
                <span>Encrypt</span>
              </button>
            </template>
          </div>

          <div class="flex items-center gap-2">
            <button
              type="button"
              @click="dedupeStore.openModal(remote.name, '', 'list', remote.type)"
              class="px-2.5 py-1 rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1.5 cursor-pointer"
              title="Launch Deduplication Workspace"
            >
              <i class="pi pi-clone text-xs"></i>
              <span>Dedupe</span>
            </button>

            <button
              @click="confirmDelete(remote.name)"
              class="text-rose-400 hover:text-rose-300 text-xs font-mono p-1 rounded hover:bg-rose-950/40 cursor-pointer ml-1"
              title="Delete remote"
            >
              <i class="pi pi-trash text-xs"></i>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Modals -->
    <ModalDedupe />
    <ModalRemoteReconnect />
    <ModalOAuthMigration />
    <ModalAuthAudit />
    <ModalHasherManager />
    <ModalCryptWrap />
  </div>
</template>
