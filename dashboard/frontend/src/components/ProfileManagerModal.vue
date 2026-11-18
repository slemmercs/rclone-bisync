<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useProfilesStore } from '../stores/profiles'
import { useRemotesStore } from '../stores/remotes'
import type { Profile, ProfileVersion } from '../types/models'
import { PROVIDER_OPTIONS_SCHEMA, type ProviderOptionSchema } from '../data/provider-options-schema'
import FlagsBuilder from './FlagsBuilder.vue'
import ModalProfileDelete from './ModalProfileDelete.vue'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ (e: 'update:visible', val: boolean): void; (e: 'open-remote-wizard'): void }>()

const profilesStore = useProfilesStore()
const remotesStore = useRemotesStore()
const profiles = computed(() => profilesStore.profiles)

const activeTab = ref<'list' | 'create' | 'edit' | 'history'>('list')
const editingProfile = ref<Partial<Profile>>({})
const excludeInput = ref('')
const formError = ref<string | null>(null)
const cliCopied = ref(false)

const historyLoading = ref(false)
const historyRecords = ref<ProfileVersion[]>([])
const showCloneDialog = ref(false)
const cloneSource = ref<Profile | null>(null)
const cloneTargetId = ref('')
const cloneTargetName = ref('')
const cloneError = ref<string | null>(null)
const showDeleteModal = ref(false)
const profileToDelete = ref<Profile | null>(null)

const normalizedRemotes = computed(() => {
  return remotesStore.remotes.map((r) => ({
    name: (r.name || '').replace(/:+$/, '').trim(),
    type: r.type || 'remote',
    rawName: r.name,
  }))
})

const activeRemoteType = computed(() => {
  const r = (editingProfile.value.remote || '').replace(/:+$/, '').trim().toLowerCase()
  if (!r || r === 'local') return 'local'
  const f = normalizedRemotes.value.find((x) => x.name.toLowerCase() === r)
  return f?.type?.toLowerCase() || 'generic'
})

const activeProviderOptions = computed<ProviderOptionSchema[]>(() => {
  return PROVIDER_OPTIONS_SCHEMA.filter((o) => o.provider === activeRemoteType.value)
})

function ensureProviderOptionsDefaults() {
  if (!editingProfile.value.flags) editingProfile.value.flags = {}
  if (!editingProfile.value.flags.provider_options) editingProfile.value.flags.provider_options = {}

  for (const opt of activeProviderOptions.value) {
    if (editingProfile.value.flags.provider_options[opt.key] === undefined) {
      if (opt.type === 'bool') {
        editingProfile.value.flags.provider_options[opt.key] = opt.default === 'true'
      } else if (opt.default) {
        editingProfile.value.flags.provider_options[opt.key] = opt.default
      }
    }
  }
}

watch(
  () => props.visible,
  (val) => {
    if (val && remotesStore.remotes.length === 0) {
      remotesStore.fetchRemotes()
    }
  },
  { immediate: true }
)

watch(activeRemoteType, () => {
  if (activeTab.value === 'create' || activeTab.value === 'edit') {
    ensureProviderOptionsDefaults()
  }
})

const synthesizedCliPreview = computed(() => {
  const p = editingProfile.value
  const local = p.local_path || '~/Sync'
  const cleanRem = (p.remote || '').replace(/:+$/, '').trim()
  const remote = cleanRem ? `${cleanRem}:${p.remote_path || ''}` : 'remote:'
  const flags = ['rclone', 'bisync', local, remote]
  if (p.flags?.compare_mode) flags.push(`--compare ${p.flags.compare_mode}`)
  if (p.flags?.conflict_resolve) flags.push(`--conflict-resolve ${p.flags.conflict_resolve}`)
  if (p.safety?.max_delete_percent !== undefined) flags.push(`--max-delete ${p.safety.max_delete_percent}%`)
  const provOpts = p.flags?.provider_options || {}
  for (const opt of activeProviderOptions.value) {
    const val = provOpts[opt.key]
    if (opt.type === 'bool' && (val === true || val === 'true')) flags.push(opt.flag)
    else if (opt.type !== 'bool' && val) flags.push(`${opt.flag}=${val}`)
  }
  if (p.flags?.custom_flags) flags.push(p.flags.custom_flags)
  return flags.join(' ')
})

function copyCli() {
  navigator.clipboard.writeText(synthesizedCliPreview.value)
  cliCopied.value = true
  setTimeout(() => { cliCopied.value = false }, 2000)
}

function openCreate() {
  const rnd = Math.random().toString(36).substring(2, 8)
  const defRem = normalizedRemotes.value[0]?.name || 'local'
  editingProfile.value = {
    id: `prof_${rnd}`,
    name: '',
    local_path: defRem !== 'local' ? `~/Sync/${defRem}` : '~/Sync',
    remote: defRem,
    remote_path: '',
    schedule: 'manual',
    review_mode: 'deletions',
    enabled: true,
    flags: { compare_mode: 'size,modtime', conflict_resolve: 'newer', custom_flags: '', provider_options: {} },
    safety: { max_delete_percent: 10, canary_protection: true },
    exclude_patterns: ['.git/**', '*.tmp', '.DS_Store'],
  }
  ensureProviderOptionsDefaults()
  excludeInput.value = '.git/**, *.tmp, .DS_Store'
  formError.value = null
  activeTab.value = 'create'
}

function openEdit(p: Profile) {
  editingProfile.value = JSON.parse(JSON.stringify(p))
  editingProfile.value.remote = (editingProfile.value.remote || '').replace(/:+$/, '').trim()
  if (!editingProfile.value.flags) editingProfile.value.flags = {}
  if (!editingProfile.value.flags.provider_options) editingProfile.value.flags.provider_options = {}

  // Ensure booleans in provider_options are actual booleans for checkbox binding
  const provOpts = editingProfile.value.flags.provider_options
  for (const opt of activeProviderOptions.value) {
    if (opt.type === 'bool') {
      const val = provOpts[opt.key]
      if (val !== undefined && val !== null) {
        provOpts[opt.key] = val === true || val === 'true' || val === '1'
      }
    }
  }

  if (!editingProfile.value.safety) {
    editingProfile.value.safety = { max_delete_percent: 10, canary_protection: true }
  } else if (editingProfile.value.safety.max_delete_percent === undefined && (editingProfile.value.safety as any).delete_threshold_pct !== undefined) {
    editingProfile.value.safety.max_delete_percent = (editingProfile.value.safety as any).delete_threshold_pct
  }

  excludeInput.value = (p.exclude_patterns || []).join(', ')
  formError.value = null
  activeTab.value = 'edit'
}

async function openHistory(p: Profile) {
  editingProfile.value = p
  activeTab.value = 'history'
  historyLoading.value = true
  try {
    historyRecords.value = await profilesStore.fetchProfileHistory(p.id)
  } catch (err: any) {
    formError.value = err.message || 'Failed'
  } finally {
    historyLoading.value = false
  }
}

async function rollbackToVersion(ver: number) {
  if (!editingProfile.value.id) return
  try {
    await profilesStore.rollbackProfile(editingProfile.value.id, ver)
    activeTab.value = 'list'
  } catch (err: any) {
    alert(err.message || 'Rollback failed')
  }
}

function openClone(p: Profile) {
  cloneSource.value = p
  cloneTargetId.value = `prof_${Math.random().toString(36).substring(2, 8)}`
  cloneTargetName.value = `${p.name} (Copy)`
  cloneError.value = null
  showCloneDialog.value = true
}

async function executeClone() {
  if (!cloneSource.value || !cloneTargetId.value.trim() || !cloneTargetName.value.trim()) return
  try {
    await profilesStore.cloneProfile(cloneSource.value.id, cloneTargetId.value.trim(), cloneTargetName.value.trim())
    showCloneDialog.value = false
  } catch (err: any) {
    cloneError.value = err.message || 'Clone failed'
  }
}

function confirmDelete(p: Profile) {
  profileToDelete.value = p
  showDeleteModal.value = true
}

async function saveProfile() {
  formError.value = null
  const p = editingProfile.value
  if (!p.name?.trim()) { formError.value = 'Profile name required'; return }
  if (!p.local_path?.trim()) { formError.value = 'Local path required'; return }
  if (!p.remote?.trim()) { formError.value = 'Storage remote required'; return }

  p.name = p.name.trim()
  p.remote = (p.remote || '').replace(/:+$/, '').trim()
  if (p.remote_path) {
    p.remote_path = p.remote_path.replace(/^:+/, '').trim()
  }
  p.exclude_patterns = excludeInput.value.split(',').map((s) => s.trim()).filter(Boolean)

  if (p.safety) {
    const pct = Number(p.safety.max_delete_percent ?? 10)
    p.safety.max_delete_percent = pct
    ;(p.safety as any).delete_threshold_pct = pct
  }

  if (p.flags?.provider_options) {
    const cleanedOpts: Record<string, string> = {}
    for (const [k, v] of Object.entries(p.flags.provider_options)) {
      if (v !== undefined && v !== null && v !== '') {
        cleanedOpts[k] = String(v)
      }
    }
    p.flags.provider_options = cleanedOpts
  }

  try {
    if (activeTab.value === 'create') await profilesStore.createProfile(p as Profile)
    else if (p.id) await profilesStore.updateProfile(p.id, p as Profile)
    activeTab.value = 'list'
  } catch (e: any) {
    formError.value = e.message || 'Failed to save'
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    header="Sync Profiles Manager"
    :style="{ width: '46rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between text-white font-bold' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <div class="flex items-center justify-between border-b border-carbon-border pb-2">
      <div class="flex items-center gap-2">
        <button
          @click="activeTab = 'list'"
          class="px-3 py-1 rounded-lg text-xs font-semibold cursor-pointer transition"
          :class="activeTab === 'list' ? 'bg-electric-blue text-white' : 'text-slate-400 hover:text-white'"
        >
          <span>Active Profiles</span>
          <span class="ml-1.5 text-[10px] px-1.5 py-0.2 rounded-full bg-carbon-elevated text-slate-300 font-mono">
            {{ profiles.length }}
          </span>
        </button>
        <button
          v-if="activeTab === 'create' || activeTab === 'edit'"
          class="px-3 py-1 rounded-lg text-xs font-semibold bg-electric-blue text-white"
        >
          {{ activeTab === 'create' ? 'Create Profile' : 'Edit Configuration' }}
        </button>
        <button
          v-if="activeTab === 'history'"
          class="px-3 py-1 rounded-lg text-xs font-semibold bg-electric-blue text-white flex items-center gap-1.5"
        >
          <i class="pi pi-history text-xs"></i><span>Version History</span>
        </button>
      </div>

      <button
        v-if="activeTab === 'list'"
        @click="openCreate()"
        class="px-3 py-1 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 cursor-pointer shadow-sm transition"
      >
        <i class="pi pi-plus text-xs"></i><span>New Profile</span>
      </button>
    </div>

    <div v-if="formError" class="p-2.5 bg-rose-500/10 border border-rose-500/20 rounded-xl text-xs text-rose-300 font-mono">
      {{ formError }}
    </div>

    <!-- TAB 1: ACTIVE PROFILES LIST -->
    <div v-if="activeTab === 'list'" class="max-h-[60vh] overflow-y-auto space-y-2.5 pr-1">
      <div v-if="profiles.length === 0" class="p-8 text-center text-slate-500 text-xs border border-dashed border-carbon-border rounded-xl">
        No synchronization profiles configured. Click "New Profile" to get started.
      </div>
      <div
        v-for="p in profiles"
        :key="p.id"
        class="p-3.5 rounded-xl border border-carbon-border bg-carbon-elevated/40 hover:border-slate-700 transition flex flex-col sm:flex-row sm:items-center justify-between gap-3"
      >
        <div class="space-y-1 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="font-bold text-sm text-white">{{ p.name }}</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded font-mono bg-carbon-elevated text-slate-400 border border-carbon-border">v{{ p.version || 1 }}</span>
            <span class="text-[9px] px-2 py-0.5 rounded font-mono font-bold uppercase bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              {{ p.state?.last_run_status || 'idle' }}
            </span>
            <span class="text-[10px] px-1.5 py-0.5 rounded font-mono bg-slate-800 text-slate-300">
              {{ p.schedule || 'manual' }}
            </span>
          </div>
          <div class="text-xs font-mono text-slate-400 flex items-center gap-1 truncate">
            <span class="truncate">{{ p.local_path }}</span>
            <span class="text-slate-600">&harr;</span>
            <span class="text-emerald-400 truncate">{{ p.remote }}{{ p.remote_path ? ':' + p.remote_path : '' }}</span>
          </div>
        </div>

        <div class="flex items-center gap-1.5 shrink-0 self-end sm:self-center">
          <button
            type="button"
            @click="profilesStore.triggerSync()"
            class="px-2.5 py-1 bg-electric-blue/10 hover:bg-electric-blue/20 text-electric-blue rounded-lg text-xs font-mono border border-electric-blue/30 cursor-pointer transition flex items-center gap-1"
            title="Trigger profile synchronization"
          >
            <i class="pi pi-play text-[10px]"></i><span>Sync</span>
          </button>
          <button
            type="button"
            @click="openHistory(p)"
            class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-300 rounded-lg text-xs font-mono border border-carbon-border cursor-pointer transition"
            title="View configuration version history & rollback"
          >
            <i class="pi pi-history text-[10px]"></i>
          </button>
          <button
            type="button"
            @click="openClone(p)"
            class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-300 rounded-lg text-xs font-mono border border-carbon-border cursor-pointer transition"
            title="Clone profile workload"
          >
            <i class="pi pi-copy text-[10px]"></i>
          </button>
          <button
            type="button"
            @click="openEdit(p)"
            class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-200 rounded-lg text-xs font-mono border border-carbon-border cursor-pointer transition"
          >
            Edit
          </button>
          <button
            type="button"
            @click="confirmDelete(p)"
            class="px-2.5 py-1 bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 rounded-lg text-xs font-mono border border-rose-500/30 cursor-pointer transition"
          >
            Delete
          </button>
        </div>
      </div>
    </div>

    <!-- TAB 2: VERSION HISTORY -->
    <div v-else-if="activeTab === 'history'" class="space-y-3 max-h-[60vh] overflow-y-auto pr-1">
      <div class="flex items-center justify-between p-3 rounded-xl bg-carbon-bg border border-carbon-border">
        <div>
          <div class="text-sm font-bold text-white">{{ editingProfile.name }}</div>
          <div class="text-xs font-mono text-slate-400">ID: {{ editingProfile.id }} • Current Version: v{{ editingProfile.version || 1 }}</div>
        </div>
        <button
          @click="activeTab = 'list'"
          class="px-3 py-1.5 bg-carbon-elevated hover:bg-carbon-border text-slate-300 rounded-lg text-xs transition cursor-pointer"
        >
          Back to Profiles
        </button>
      </div>

      <div v-if="historyLoading" class="p-8 text-center text-electric-blue text-xs animate-pulse">
        Loading historical snapshots...
      </div>
      <div v-else-if="historyRecords.length === 0" class="p-8 text-center text-slate-500 text-xs border border-dashed border-carbon-border rounded-xl">
        No version history recorded yet. Historical snapshots are archived each time profile settings are modified.
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="rec in historyRecords"
          :key="rec.version"
          class="p-3 rounded-xl bg-carbon-bg border border-carbon-border flex items-center justify-between gap-3 text-xs"
        >
          <div class="space-y-0.5 min-w-0">
            <div class="flex items-center gap-2">
              <span class="font-mono font-bold text-indigo-400">v{{ rec.version }}</span>
              <span class="text-slate-500">•</span>
              <span class="text-slate-300 font-medium">{{ rec.reason || 'Configuration update' }}</span>
              <span v-if="rec.version === editingProfile.version" class="text-[9px] px-1.5 py-0.2 rounded font-bold uppercase bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Current</span>
            </div>
            <div class="text-[11px] text-slate-400 font-mono flex items-center gap-2">
              <span>{{ new Date(rec.timestamp).toLocaleString() }}</span>
              <span>•</span>
              <span class="text-slate-500">{{ rec.snapshot.remote }} &harr; {{ rec.snapshot.local_path }}</span>
            </div>
          </div>
          <button
            v-if="rec.version !== editingProfile.version"
            type="button"
            @click="rollbackToVersion(rec.version)"
            class="px-3 py-1 bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 rounded-lg text-xs font-semibold border border-amber-500/30 transition cursor-pointer shrink-0"
          >
            Rollback to v{{ rec.version }}
          </button>
        </div>
      </div>
    </div>

    <!-- TAB 3: CREATE / EDIT PROFILE FORM -->
    <div v-else class="space-y-4 max-h-[60vh] overflow-y-auto pr-1 text-xs">
      <!-- Identity & Remote Target -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label class="block text-slate-300 font-semibold mb-1">Profile Name <span class="text-rose-400">*</span></label>
          <input v-model="editingProfile.name" type="text" placeholder="e.g. Work Drive Sync" class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-electric-blue" />
        </div>
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="block text-slate-300 font-semibold">Profile ID</label>
            <span class="text-[10px] text-slate-500 font-mono">{{ activeTab === 'edit' ? 'immutable' : 'auto-generated' }}</span>
          </div>
          <input v-model="editingProfile.id" type="text" :disabled="activeTab === 'edit'" class="w-full bg-carbon-bg border border-carbon-border disabled:opacity-50 rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label class="block text-slate-300 font-semibold mb-1">Storage Remote <span class="text-rose-400">*</span></label>
          <select
            v-if="normalizedRemotes.length > 0"
            v-model="editingProfile.remote"
            @change="editingProfile.remote = ($event.target as HTMLSelectElement).value.replace(/:+$/, '').trim()"
            class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option v-for="r in normalizedRemotes" :key="r.name" :value="r.name">{{ r.name }} ({{ r.type }})</option>
          </select>
          <input
            v-else
            v-model="editingProfile.remote"
            @blur="editingProfile.remote = (editingProfile.remote || '').replace(/:+$/, '').trim()"
            type="text"
            placeholder="e.g. gdrive"
            class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <div class="mt-1 flex items-center justify-between text-[11px]">
            <span class="text-slate-500">Need a new target?</span>
            <button type="button" @click="emit('open-remote-wizard')" class="text-electric-blue hover:underline cursor-pointer">+ Provision Remote</button>
          </div>
        </div>
        <div>
          <label class="block text-slate-300 font-semibold mb-1">Remote Subpath (Optional)</label>
          <input v-model="editingProfile.remote_path" type="text" placeholder="e.g. Documents" class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
      </div>

      <div>
        <label class="block text-slate-300 font-semibold mb-1">Local Directory Path <span class="text-rose-400">*</span></label>
        <input v-model="editingProfile.local_path" type="text" placeholder="e.g. ~/Sync/Documents" class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
      </div>

      <!-- Core Sync & Comparison Settings Grid -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 p-3 bg-carbon-surface/60 border border-carbon-border rounded-xl">
        <div>
          <label class="block text-[11px] text-slate-400 font-semibold mb-1">Schedule</label>
          <select v-model="editingProfile.schedule" class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue">
            <option value="manual">Manual</option>
            <option value="15m">15 min</option>
            <option value="30m">30 min</option>
            <option value="1h">1 hour</option>
            <option value="4h">4 hours</option>
            <option value="daily">Daily</option>
          </select>
        </div>
        <div>
          <label class="block text-[11px] text-slate-400 font-semibold mb-1">Review</label>
          <select v-model="editingProfile.review_mode" class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue">
            <option value="disabled">Disabled</option>
            <option value="deletions">Deletions</option>
            <option value="conflicts">Conflicts</option>
            <option value="all">All</option>
          </select>
        </div>
        <div>
          <label class="block text-[11px] text-slate-400 font-semibold mb-1">Compare</label>
          <select v-model="editingProfile.flags!.compare_mode" class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue">
            <option value="size,modtime">Size & ModTime</option>
            <option value="size,modtime,checksum">Size, ModTime, Hash</option>
            <option value="checksum">Hash Only</option>
            <option value="modtime">ModTime Only</option>
          </select>
        </div>
        <div>
          <label class="block text-[11px] text-slate-400 font-semibold mb-1">Conflict</label>
          <select v-model="editingProfile.flags!.conflict_resolve" class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue">
            <option value="newer">Newer Wins</option>
            <option value="older">Older Wins</option>
            <option value="larger">Larger Wins</option>
            <option value="smaller">Smaller Wins</option>
            <option value="path1">Local Wins</option>
            <option value="path2">Remote Wins</option>
          </select>
        </div>
      </div>

      <!-- Safety & Excludes -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3 bg-carbon-surface/60 border border-carbon-border rounded-xl">
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="text-[11px] text-slate-400 font-semibold">Max Delete Threshold (%)</label>
            <span class="text-[10px] text-slate-500 font-mono">{{ editingProfile.safety?.max_delete_percent || 10 }}%</span>
          </div>
          <input v-model.number="editingProfile.safety!.max_delete_percent" type="number" min="0" max="100" class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
        <div>
          <label class="block text-[11px] text-slate-400 font-semibold mb-1">Exclude Patterns</label>
          <input v-model="excludeInput" type="text" placeholder=".git/**, *.tmp, .DS_Store" class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
      </div>

      <!-- Dynamic Target Provider Options (Tier 1 Providers) -->
      <div v-if="activeProviderOptions.length > 0" class="p-3 bg-carbon-surface/60 border border-carbon-border rounded-xl space-y-2.5">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <label class="text-xs font-semibold text-slate-200">Target Provider Options</label>
            <span class="text-[9px] px-2 py-0.5 rounded font-mono font-bold uppercase bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              {{ activeRemoteType }}
            </span>
          </div>
          <span class="text-[10px] text-slate-500 font-mono">{{ editingProfile.remote }}</span>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
          <label
            v-for="opt in activeProviderOptions.filter((o) => o.type === 'bool')"
            :key="opt.key"
            class="flex items-center gap-2 p-2 bg-carbon-bg rounded-lg border border-carbon-border cursor-pointer hover:border-slate-600 transition"
            :title="opt.desc"
          >
            <input
              type="checkbox"
              v-model="editingProfile.flags!.provider_options![opt.key]"
              class="rounded bg-carbon-surface border-carbon-border text-electric-blue"
            />
            <span class="text-[11px] text-slate-300 font-medium">{{ opt.label }}</span>
          </label>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <div
            v-for="opt in activeProviderOptions.filter((o) => o.type !== 'bool')"
            :key="opt.key"
          >
            <label class="block text-[10px] font-semibold text-slate-400 mb-1">{{ opt.label }}</label>
            <select
              v-if="opt.type === 'select'"
              v-model="editingProfile.flags!.provider_options![opt.key]"
              class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
            >
              <option v-for="c in opt.options" :key="c" :value="c">{{ c }}</option>
            </select>
            <input
              v-else
              v-model="editingProfile.flags!.provider_options![opt.key]"
              type="text"
              :placeholder="opt.placeholder || ''"
              class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
            />
          </div>
        </div>
      </div>

      <!-- Guided Flags Builder -->
      <FlagsBuilder
        v-model="editingProfile.flags!.custom_flags"
        :remote-type="activeRemoteType"
      />

      <!-- Synthesized Real-Time CLI Preview -->
      <div class="p-3 bg-carbon-bg border border-carbon-border rounded-xl space-y-1.5">
        <div class="flex items-center justify-between text-[11px] text-slate-400">
          <span class="font-semibold text-slate-300">Live Compiled CLI Preview</span>
          <button type="button" @click="copyCli()" class="text-electric-blue hover:underline cursor-pointer flex items-center gap-1 font-mono">
            <i :class="cliCopied ? 'pi pi-check text-emerald-400' : 'pi pi-copy text-xs'"></i>
            <span>{{ cliCopied ? 'Copied!' : 'Copy CLI' }}</span>
          </button>
        </div>
        <div class="font-mono text-[11px] p-2 bg-carbon-surface rounded-lg border border-carbon-border text-emerald-300 break-all select-all">
          {{ synthesizedCliPreview }}
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center justify-end gap-2 pt-2 border-t border-carbon-border">
        <button type="button" @click="activeTab = 'list'" class="px-3 py-1.5 bg-carbon-elevated text-slate-300 rounded-lg text-xs cursor-pointer hover:bg-carbon-border transition">Cancel</button>
        <button type="button" @click="saveProfile()" class="px-4 py-1.5 bg-electric-blue hover:bg-electric-blue/90 text-white rounded-lg text-xs font-semibold cursor-pointer transition shadow-sm">
          Save Configuration
        </button>
      </div>
    </div>

    <!-- CLONE PROFILE DIALOG -->
    <Dialog
      v-model:visible="showCloneDialog"
      modal
      header="Clone Synchronization Workload"
      :style="{ width: '28rem', maxWidth: '95vw' }"
      :pt="{
        root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-4' },
        header: { class: 'border-b border-carbon-border pb-2 text-white font-bold' },
        content: { class: 'pt-3 space-y-3' },
      }"
    >
      <div v-if="cloneError" class="p-2.5 bg-rose-500/10 border border-rose-500/20 rounded-lg text-xs text-rose-300 font-mono">
        {{ cloneError }}
      </div>
      <div>
        <label class="block text-slate-400 mb-1 font-semibold">Target Profile Name</label>
        <input v-model="cloneTargetName" type="text" class="w-full bg-carbon-bg border border-carbon-border rounded-lg p-2 text-white" />
      </div>
      <div>
        <label class="block text-slate-400 mb-1 font-semibold">Target Profile ID</label>
        <input v-model="cloneTargetId" type="text" class="w-full bg-carbon-bg border border-carbon-border rounded-lg p-2 text-white font-mono" />
      </div>
      <div class="flex items-center justify-end gap-2 pt-2 border-t border-carbon-border">
        <button @click="showCloneDialog = false" class="px-3 py-1.5 bg-carbon-elevated text-slate-300 rounded-lg text-xs cursor-pointer">Cancel</button>
        <button @click="executeClone()" class="px-3.5 py-1.5 bg-electric-blue text-white rounded-lg text-xs font-semibold cursor-pointer">Clone Profile</button>
      </div>
    </Dialog>

    <!-- DELETE PROFILE SAFEGUARD MODAL -->
    <ModalProfileDelete
      v-model:visible="showDeleteModal"
      :profile="profileToDelete"
      @deleted="activeTab = 'list'"
    />
  </Dialog>
</template>
