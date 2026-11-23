<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRemotesStore } from '../stores/remotes'
import { useProfilesStore } from '../stores/profiles'
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
const profilesStore = useProfilesStore()
const dedupeStore = useDedupeStore()

const categoryFilter = ref<'all' | 'cloud' | 'object' | 'protocol' | 'virtual'>('all')
const searchQuery = ref<string>('')

const remotes = computed(() => remotesStore.remotes || [])
const summary = computed(() => remotesStore.summary)
const loading = computed(() => remotesStore.loading)

const activeRemoteName = computed<string | null>(() => {
  const remote = profilesStore.activeProfile?.remote
  if (!remote) return null
  return remote.replace(/:.*$/, '').trim() || null
})

const compositeTypes = new Set(['combine', 'union'])
const virtualTypes = new Set(['crypt', 'hasher', 'alias'])

// Adjacency map: parent remote name -> list of child overlay remotes
const childrenByParent = computed<Record<string, any[]>>(() => {
  const map: Record<string, any[]> = {}
  const list = remotes.value

  for (const r of list) {
    let parent = r.underlying_remote
    if (parent) {
      if (parent.endsWith(':')) {
        parent = parent.slice(0, -1)
      }
      parent = parent.trim()
      if (parent) {
        if (!map[parent]) {
          map[parent] = []
        }
        if (!map[parent].some((existing: any) => existing.name === r.name)) {
          map[parent].push(r)
        }
      }
    }
  }

  // Also include bidirectional references from crypt_overlay / hasher_overlay / overlays
  for (const r of list) {
    const overlayNames: string[] = []
    if (r.crypt_overlay) overlayNames.push(r.crypt_overlay)
    if (r.hasher_overlay) overlayNames.push(r.hasher_overlay)
    if (Array.isArray(r.overlays)) {
      for (const ov of r.overlays) {
        if (typeof ov === 'string' && !overlayNames.includes(ov)) {
          overlayNames.push(ov)
        }
      }
    }

    if (overlayNames.length > 0) {
      if (!map[r.name]) {
        map[r.name] = []
      }
      for (const ovName of overlayNames) {
        const found = list.find((item: any) => item.name === ovName)
        if (found && !map[r.name].some((existing: any) => existing.name === found.name)) {
          map[r.name].push(found)
        }
      }
    }
  }

  return map
})

// Set of remote names that are cleanly nested under a recognized parent
const childRemoteNames = computed<Set<string>>(() => {
  const set = new Set<string>()
  const list = remotes.value
  const allNames = new Set(list.map((r: any) => r.name))

  for (const r of list) {
    const type = (r.type || '').toLowerCase()
    let parent = r.underlying_remote
    if (parent && parent.endsWith(':')) {
      parent = parent.slice(0, -1)
    }
    parent = (parent || '').trim()

    if (virtualTypes.has(type) && parent && allNames.has(parent)) {
      set.add(r.name)
    }
  }

  for (const [parentName, children] of Object.entries(childrenByParent.value)) {
    if (allNames.has(parentName)) {
      for (const c of children) {
        set.add(c.name)
      }
    }
  }

  return set
})

// Root remotes: physical base remotes or top-level containers
const rootRemotes = computed(() => {
  return remotes.value.filter((r: any) => {
    const type = (r.type || '').toLowerCase()
    if (compositeTypes.has(type)) return false
    if (childRemoteNames.value.has(r.name)) return false
    return true
  })
})

// Composite (union, combine) or orphan virtual remotes without a valid parent
const compositeOrOrphanRemotes = computed(() => {
  const rootNames = new Set(rootRemotes.value.map((r: any) => r.name))
  return remotes.value.filter((r: any) => {
    if (rootNames.has(r.name)) return false
    if (childRemoteNames.value.has(r.name)) return false

    const type = (r.type || '').toLowerCase()
    if (compositeTypes.has(type)) return true
    if (virtualTypes.has(type)) return true
    return false
  })
})

function getDescendantRemotes(remoteName: string, map: Record<string, any[]>): any[] {
  const result: any[] = []
  const queue = [...(map[remoteName] || [])]
  const visited = new Set<string>()
  while (queue.length > 0) {
    const current = queue.shift()!
    if (visited.has(current.name)) continue
    visited.add(current.name)
    result.push(current)
    if (map[current.name]) {
      queue.push(...map[current.name])
    }
  }
  return result
}

function matchesSearchAndCategory(remote: any, allDescendants: any[]): boolean {
  if (categoryFilter.value !== 'all') {
    const familyTypes = [remote, ...allDescendants].map((r) => (r.type || '').toLowerCase())

    if (categoryFilter.value === 'cloud') {
      const cloudTypes = ['drive', 'onedrive', 'dropbox', 'box', 'mega', 'pcloud']
      if (!cloudTypes.some((t) => familyTypes.includes(t))) return false
    } else if (categoryFilter.value === 'object') {
      const objTypes = ['s3', 'b2', 'gcs', 'azureblob', 'wasabi', 'minio']
      if (!objTypes.some((t) => familyTypes.includes(t))) return false
    } else if (categoryFilter.value === 'protocol') {
      const protoTypes = ['sftp', 'ftp', 'webdav', 'smb', 'http']
      if (!protoTypes.some((t) => familyTypes.includes(t))) return false
    } else if (categoryFilter.value === 'virtual') {
      const virtTypes = ['hasher', 'crypt', 'alias', 'combine', 'chunker', 'union']
      if (!virtTypes.some((t) => familyTypes.includes(t))) return false
    }
  }

  const q = searchQuery.value.toLowerCase().trim()
  if (q) {
    const familyStrings = [remote, ...allDescendants].map(
      (r) => `${r.name || ''} ${r.type || ''} ${r.endpoint || ''} ${r.underlying_remote || ''}`.toLowerCase()
    )
    if (!familyStrings.some((s) => s.includes(q))) return false
  }

  return true
}

const filteredRootRemotes = computed(() => {
  return rootRemotes.value.filter((r: any) => {
    const descendants = getDescendantRemotes(r.name, childrenByParent.value)
    return matchesSearchAndCategory(r, descendants)
  })
})

const filteredCompositeRemotes = computed(() => {
  return compositeOrOrphanRemotes.value.filter((r: any) => {
    const descendants = getDescendantRemotes(r.name, childrenByParent.value)
    return matchesSearchAndCategory(r, descendants)
  })
})

function checkAuth(name?: string) {
  remotesStore.checkAuth(name)
}

function confirmDelete(name: string) {
  const target = remotes.value.find((r: any) => r.name === name)
  let warning = `Permanently delete remote configuration "${name}"? Active sync profiles referencing this remote will be impacted.`

  const descendants = getDescendantRemotes(name, childrenByParent.value)
  if (descendants.length > 0) {
    const childNames = descendants.map((d: any) => `"${d.name}" (${d.type || 'overlay'})`).join(', ')
    warning += `\n\n⚠️ CASCADE WARNING: This remote has ${descendants.length} dependent overlay(s): ${childNames}. Deleting it will orphan or break these downstream overlays.`
  }

  if (target && (target.type || '').toLowerCase() === 'crypt') {
    const under = target.underlying_remote ? ` on base remote "${target.underlying_remote}"` : ''
    warning += `\n\n⚠️ IMPORTANT: Deleting this crypt remote only removes local decryption keys. Encrypted cipher files stored in the cloud${under} will remain unreadable ciphertext unless decrypted prior to deletion.`
  }

  if (confirm(warning)) {
    remotesStore.deleteRemote(name)
  }
}

onMounted(() => {
  remotesStore.fetchRemotes()
  if (profilesStore.profiles.length === 0) {
    profilesStore.fetchProfiles()
  }
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
    <div
      v-if="filteredRootRemotes.length === 0 && filteredCompositeRemotes.length === 0"
      class="p-12 text-center text-slate-500 bg-carbon-surface border border-carbon-border rounded-2xl"
    >
      <i class="pi pi-cloud text-4xl text-slate-600 mb-3 block"></i>
      <p class="text-sm font-semibold text-slate-400">No matching storage remotes found</p>
    </div>

    <template v-else>
      <!-- Base Remotes & Clustered Pipelines -->
      <div v-if="filteredRootRemotes.length > 0" class="space-y-3">
        <div class="flex items-center justify-between px-1">
          <div class="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span class="font-bold text-slate-200 uppercase tracking-wider text-[11px]">Storage Endpoints & Overlay Pipelines</span>
            <span class="text-[10px] text-slate-500">({{ filteredRootRemotes.length }})</span>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <RemoteCard
            v-for="remote in filteredRootRemotes"
            :key="remote.name"
            :remote="remote"
            :children-by-parent="childrenByParent"
            :active-remote-name="activeRemoteName"
            @delete="confirmDelete"
          />
        </div>
      </div>

      <!-- Composite & Independent Overlays Section -->
      <div
        v-if="filteredCompositeRemotes.length > 0"
        class="space-y-3 pt-6 border-t border-carbon-border/60"
      >
        <div class="flex items-center justify-between px-1">
          <div class="flex items-center gap-2 text-xs font-mono text-slate-400">
            <i class="pi pi-th-large text-electric-blue text-xs"></i>
            <span class="font-bold text-slate-200 uppercase tracking-wider text-[11px]">Composite & Independent Overlays</span>
            <span class="text-[10px] text-slate-500">({{ filteredCompositeRemotes.length }})</span>
          </div>
          <span class="text-[10px] text-slate-500 font-mono">Multi-backend or unlinked virtual targets</span>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <RemoteCard
            v-for="remote in filteredCompositeRemotes"
            :key="remote.name"
            :remote="remote"
            :children-by-parent="childrenByParent"
            :active-remote-name="activeRemoteName"
            @delete="confirmDelete"
          />
        </div>
      </div>
    </template>

    <!-- Modals -->
    <ModalDedupe />
    <ModalRemoteReconnect />
    <ModalOAuthMigration />
    <ModalAuthAudit />
    <ModalHasherManager />
    <ModalCryptWrap />
  </div>
</template>
