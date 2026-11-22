<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRemotesStore } from '../stores/remotes'
import { useDedupeStore } from '../stores/dedupe'
import RemotesKpiRibbon from '../components/remotes/RemotesKpiRibbon.vue'
import RemoteCard from '../components/remotes/RemoteCard.vue'
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

const filteredRemotes = computed(() => {
  let list = remotes.value || []

  if (categoryFilter.value === 'cloud') {
    list = list.filter((r) =>
      ['drive', 'onedrive', 'dropbox', 'box', 'mega', 'pcloud'].includes((r.type || '').toLowerCase())
    )
  } else if (categoryFilter.value === 'object') {
    list = list.filter((r) =>
      ['s3', 'b2', 'gcs', 'azureblob', 'wasabi', 'minio'].includes((r.type || '').toLowerCase())
    )
  } else if (categoryFilter.value === 'protocol') {
    list = list.filter((r) =>
      ['sftp', 'ftp', 'webdav', 'smb', 'http'].includes((r.type || '').toLowerCase())
    )
  } else if (categoryFilter.value === 'virtual') {
    list = list.filter((r) =>
      ['hasher', 'crypt', 'alias', 'combine', 'chunker', 'union'].includes((r.type || '').toLowerCase())
    )
  }

  const q = searchQuery.value.toLowerCase().trim()
  if (q) {
    list = list.filter(
      (r) => (r.name || '').toLowerCase().includes(q) || (r.type || '').toLowerCase().includes(q)
    )
  }

  return list
})

function checkAuth(name?: string) {
  remotesStore.checkAuth(name)
}

function confirmDelete(name: string) {
  if (
    confirm(
      `Permanently delete remote configuration "${name}"? Active sync profiles referencing this remote will be impacted.`
    )
  ) {
    remotesStore.deleteRemote(name)
  }
}

onMounted(() => {
  remotesStore.fetchRemotes()
})
</script>

<template>
  <div class="space-y-6">
    <RemotesKpiRibbon :summary="summary" :remotes-count="remotes.length" />

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
          @click="remotesStore.openWizard()"
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
      <RemoteCard
        v-for="remote in filteredRemotes"
        :key="remote.name"
        :remote="remote"
        @delete="confirmDelete"
      />

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
