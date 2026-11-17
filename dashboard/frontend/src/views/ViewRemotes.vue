<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRemotesStore } from '../stores/remotes'
import { useDedupeStore } from '../stores/dedupe'
import ModalDedupe from '../components/ModalDedupe.vue'
import ModalRemoteWizard from '../components/ModalRemoteWizard.vue'

const remotesStore = useRemotesStore()
const dedupeStore = useDedupeStore()

const categoryFilter = ref<'all' | 'cloud' | 'object' | 'protocol'>('all')
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
        <div class="space-y-2">
          <div class="flex items-start justify-between">
            <div>
              <h4 class="font-bold text-white text-sm font-mono flex items-center gap-2">
                <span>{{ remote.name }}</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded uppercase bg-carbon-elevated text-slate-400 border border-carbon-border font-mono">
                  {{ remote.type || 'remote' }}
                </span>
              </h4>
              <div class="text-[11px] text-slate-400 font-mono mt-1">
                {{ remote.endpoint || remote.root || 'Configured storage backend' }}
              </div>
            </div>
            <span
              class="text-[10px] font-mono px-2 py-0.5 rounded-full font-bold uppercase"
              :class="remote.status === 'error' ? 'bg-rose-500/10 text-rose-400 border border-rose-500/30' : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'"
            >
              {{ remote.status === 'error' ? 'ERROR' : 'CONNECTED' }}
            </span>
          </div>
        </div>

        <div class="pt-3 border-t border-carbon-border flex items-center justify-between text-xs font-mono">
          <button
            type="button"
            @click="dedupeStore.openModal(remote.name)"
            class="px-2.5 py-1 rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1.5 cursor-pointer"
            title="Launch Deduplication Workspace"
          >
            <i class="pi pi-clone text-xs"></i>
            <span>Deduplicate</span>
          </button>

          <div class="flex items-center gap-2">
            <button
              @click="checkAuth(remote.name)"
              class="text-electric-blue hover:text-sky-300 text-xs font-mono flex items-center gap-1 cursor-pointer"
              title="Test connection authentication"
            >
              <i class="pi pi-check-circle text-xs"></i>
              <span>Test Auth</span>
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

    <!-- Deduplication & Remote Wizard Modals -->
    <ModalDedupe />
    <ModalRemoteWizard />
  </div>
</template>
