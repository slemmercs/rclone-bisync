<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRemotesStore } from '../../stores/remotes'
import type { Profile } from '../../types/models'
import { PROVIDER_OPTIONS_SCHEMA, type ProviderOptionSchema } from '../../data/provider-options-schema'
import FlagsBuilder from '../FlagsBuilder.vue'

const props = defineProps<{
  isCreate: boolean
  profile: Partial<Profile>
  normalizedRemotes: Array<{
    name: string
    type: string
    rawName?: string
    underlying_remote?: string
    underlying_path?: string
    crypt_overlay?: string
    hasher_overlay?: string
  }>
}>()

const emit = defineEmits<{
  (e: 'save', profile: Partial<Profile>, excludeInput: string): void
  (e: 'cancel'): void
  (e: 'open-remote-wizard'): void
}>()

const remotesStore = useRemotesStore()

const editingProfile = ref<Partial<Profile>>(JSON.parse(JSON.stringify(props.profile)))
const excludeInput = ref((props.profile.exclude_patterns || []).join(', '))
const cliCopied = ref(false)

const selectedRemoteInfo = computed(() => {
  const r = (editingProfile.value.remote || '').replace(/:+$/, '').trim().toLowerCase()
  if (!r || r === 'local') return null
  return remotesStore.remotes.find((x) => (x.name || '').toLowerCase() === r) || null
})

const selectedRemotePipeline = computed(() => {
  const r = (editingProfile.value.remote || '').replace(/:+$/, '').trim()
  if (!r || r.toLowerCase() === 'local') {
    return { isEncrypted: false, isAccelerated: false, chain: [] as string[] }
  }

  let isEncrypted = false
  let isAccelerated = false
  const chain: string[] = [r]
  let currentName = r
  const visited = new Set<string>([r.toLowerCase()])

  while (currentName) {
    const entry = (remotesStore.remotes || []).find(
      (x) => (x.name || '').toLowerCase() === currentName.toLowerCase()
    )
    if (!entry) break
    const rtype = (entry.type || '').toLowerCase()
    if (rtype === 'crypt') isEncrypted = true
    if (rtype === 'hasher') isAccelerated = true

    if (entry.underlying_remote && !visited.has(entry.underlying_remote.toLowerCase())) {
      chain.push(entry.underlying_remote)
      visited.add(entry.underlying_remote.toLowerCase())
      currentName = entry.underlying_remote
    } else {
      break
    }
  }

  return { isEncrypted, isAccelerated, chain }
})

const originalRemoteInfo = computed(() => {
  const orig = (props.profile.remote || '').replace(/:+$/, '').trim().toLowerCase()
  if (!orig || orig === 'local') return null
  return remotesStore.remotes.find((x) => (x.name || '').toLowerCase() === orig) || null
})

const isSwitchingEncryption = computed(() => {
  if (props.isCreate) return false
  const origType = (originalRemoteInfo.value?.type || '').toLowerCase()
  const newType = (selectedRemoteInfo.value?.type || '').toLowerCase()
  const origEncrypted = origType === 'crypt'
  const newEncrypted = newType === 'crypt'
  return origEncrypted !== newEncrypted
})

watch(
  () => props.profile,
  (newVal) => {
    editingProfile.value = JSON.parse(JSON.stringify(newVal))
    excludeInput.value = (newVal.exclude_patterns || []).join(', ')
    ensureProviderOptionsDefaults()
  },
  { deep: true }
)

const activeRemoteType = computed(() => {
  const r = (editingProfile.value.remote || '').replace(/:+$/, '').trim().toLowerCase()
  if (!r || r === 'local') return 'local'
  const f = props.normalizedRemotes.find((x) => x.name.toLowerCase() === r)
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

watch(activeRemoteType, () => {
  ensureProviderOptionsDefaults()
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
  setTimeout(() => {
    cliCopied.value = false
  }, 2000)
}

function handleSave() {
  emit('save', editingProfile.value, excludeInput.value)
}
</script>

<template>
  <div class="space-y-4 max-h-[60vh] overflow-y-auto pr-1 text-xs">
    <!-- Identity & Remote Target -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <div>
        <label class="block text-slate-300 font-semibold mb-1">Profile Name <span class="text-rose-400">*</span></label>
        <input
          v-model="editingProfile.name"
          type="text"
          placeholder="e.g. Work Drive Sync"
          class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-electric-blue"
        />
      </div>
      <div>
        <div class="flex items-center justify-between mb-1">
          <label class="block text-slate-300 font-semibold">Profile ID</label>
          <span class="text-[10px] text-slate-500 font-mono">{{ !isCreate ? 'immutable' : 'auto-generated' }}</span>
        </div>
        <input
          v-model="editingProfile.id"
          type="text"
          :disabled="!isCreate"
          class="w-full bg-carbon-bg border border-carbon-border disabled:opacity-50 rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
        />
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
          <button
            type="button"
            @click="emit('open-remote-wizard')"
            class="text-electric-blue hover:underline cursor-pointer"
          >
            + Provision Remote
          </button>
        </div>
      </div>
      <div>
        <label class="block text-slate-300 font-semibold mb-1">Remote Subpath (Optional)</label>
        <input
          v-model="editingProfile.remote_path"
          type="text"
          placeholder="e.g. Documents"
          class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
        />
      </div>
    </div>

    <!-- Encryption Switch Warning -->
    <div
      v-if="isSwitchingEncryption"
      class="p-3 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-300 space-y-1"
    >
      <div class="flex items-center gap-1.5 font-bold text-amber-200">
        <i class="pi pi-exclamation-triangle text-amber-400"></i>
        <span>Switching Encryption Target Requires Resync (--resync)</span>
      </div>
      <p class="text-[11px] leading-relaxed text-amber-300/90 font-sans">
        You are changing between an encrypted (Crypt) and an unencrypted remote target. Existing bisync state listings will not match. A full <strong>--resync</strong> is required after saving to re-index directory state safely.
      </p>
    </div>

    <!-- Selected Remote Overlay Pipeline Preview -->
    <div
      v-if="selectedRemotePipeline.chain.length > 1"
      class="p-2.5 rounded-xl bg-carbon-surface/80 border border-carbon-border/70 flex items-center justify-between gap-3 text-xs"
    >
      <div class="flex items-center gap-2">
        <span
          v-if="selectedRemotePipeline.isEncrypted"
          class="px-1.5 py-0.5 rounded font-mono font-bold uppercase text-[9px] bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 flex items-center gap-1"
        >
          <i class="pi pi-shield text-[9px]"></i> Encrypted
        </span>
        <span
          v-if="selectedRemotePipeline.isAccelerated"
          class="px-1.5 py-0.5 rounded font-mono font-bold uppercase text-[9px] bg-violet-500/10 text-violet-400 border border-violet-500/20 flex items-center gap-1"
        >
          <i class="pi pi-bolt text-[9px]"></i> Accelerated
        </span>
        <span class="text-[11px] text-slate-400 font-sans">Pipeline:</span>
      </div>
      <div class="flex items-center gap-1.5 font-mono text-[11px] overflow-x-auto">
        <span class="px-1.5 py-0.2 rounded bg-slate-800 text-slate-300 border border-slate-700 text-[10px]">local</span>
        <i class="pi pi-arrow-right text-[8px] text-slate-500"></i>
        <template v-for="(node, idx) in selectedRemotePipeline.chain" :key="node">
          <span
            class="px-1.5 py-0.2 rounded border text-[10px]"
            :class="idx === 0 && selectedRemotePipeline.isEncrypted ? 'bg-cyan-950/60 text-cyan-300 border-cyan-800' : idx === 0 && selectedRemotePipeline.isAccelerated ? 'bg-violet-950/60 text-violet-300 border-violet-800' : 'bg-slate-900 text-slate-400 border-slate-800'"
          >
            {{ node }}
          </span>
          <i v-if="idx < selectedRemotePipeline.chain.length - 1" class="pi pi-arrow-right text-[8px] text-slate-500"></i>
        </template>
      </div>
    </div>

    <!-- Hint if plain base remote selected and crypt overlay is available -->
    <div
      v-if="!selectedRemotePipeline.isEncrypted && selectedRemoteInfo?.crypt_overlay"
      class="text-[11px] text-cyan-400/90 flex items-center gap-1.5 font-sans px-1"
    >
      <i class="pi pi-info-circle text-xs shrink-0"></i>
      <span>Tip: Encrypted overlay <strong>{{ selectedRemoteInfo.crypt_overlay }}</strong> is configured for this remote. Select it to bisync encrypted ciphertext.</span>
    </div>

    <div>
      <label class="block text-slate-300 font-semibold mb-1">Local Directory Path <span class="text-rose-400">*</span></label>
      <input
        v-model="editingProfile.local_path"
        type="text"
        placeholder="e.g. ~/Sync/Documents"
        class="w-full bg-carbon-bg border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
      />
    </div>

    <!-- Core Sync & Comparison Settings Grid -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 p-3 bg-carbon-surface/60 border border-carbon-border rounded-xl">
      <div>
        <label class="block text-[11px] text-slate-400 font-semibold mb-1">Schedule</label>
        <select
          v-model="editingProfile.schedule"
          class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue"
        >
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
        <select
          v-model="editingProfile.review_mode"
          class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue"
        >
          <option value="disabled">Disabled</option>
          <option value="deletions">Deletions</option>
          <option value="conflicts">Conflicts</option>
          <option value="all">All</option>
        </select>
      </div>
      <div>
        <label class="block text-[11px] text-slate-400 font-semibold mb-1">Compare</label>
        <select
          v-if="editingProfile.flags"
          v-model="editingProfile.flags.compare_mode"
          class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue"
        >
          <option value="size,modtime">Size & ModTime</option>
          <option value="size,modtime,checksum">Size, ModTime, Hash</option>
          <option value="checksum">Hash Only</option>
          <option value="modtime">ModTime Only</option>
        </select>
      </div>
      <div>
        <label class="block text-[11px] text-slate-400 font-semibold mb-1">Conflict</label>
        <select
          v-if="editingProfile.flags"
          v-model="editingProfile.flags.conflict_resolve"
          class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white focus:outline-none focus:border-electric-blue"
        >
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
        <input
          v-if="editingProfile.safety"
          v-model.number="editingProfile.safety.max_delete_percent"
          type="number"
          min="0"
          max="100"
          class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
        />
      </div>
      <div>
        <label class="block text-[11px] text-slate-400 font-semibold mb-1">Exclude Patterns</label>
        <input
          v-model="excludeInput"
          type="text"
          placeholder=".git/**, *.tmp, .DS_Store"
          class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
        />
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
            v-if="editingProfile.flags?.provider_options"
            type="checkbox"
            v-model="editingProfile.flags.provider_options[opt.key]"
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
            v-if="opt.type === 'select' && editingProfile.flags?.provider_options"
            v-model="editingProfile.flags.provider_options[opt.key]"
            class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option v-for="c in opt.options" :key="c" :value="c">{{ c }}</option>
          </select>
          <input
            v-else-if="editingProfile.flags?.provider_options"
            v-model="editingProfile.flags.provider_options[opt.key]"
            type="text"
            :placeholder="opt.placeholder || ''"
            class="w-full bg-carbon-bg border border-carbon-border rounded-md px-2 py-1 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
      </div>
    </div>

    <!-- Guided Flags Builder -->
    <FlagsBuilder
      v-if="editingProfile.flags"
      v-model="editingProfile.flags.custom_flags"
      :remote-type="activeRemoteType"
    />

    <!-- Synthesized Real-Time CLI Preview -->
    <div class="p-3 bg-carbon-bg border border-carbon-border rounded-xl space-y-1.5">
      <div class="flex items-center justify-between text-[11px] text-slate-400">
        <span class="font-semibold text-slate-300">Live Compiled CLI Preview</span>
        <button
          type="button"
          @click="copyCli()"
          class="text-electric-blue hover:underline cursor-pointer flex items-center gap-1 font-mono"
        >
          <i :class="cliCopied ? 'pi pi-check text-emerald-400' : 'pi pi-copy text-xs'"></i>
          <span>{{ cliCopied ? 'Copied!' : 'Copy CLI' }}</span>
        </button>
      </div>
      <div class="font-mono text-[11px] p-2 bg-carbon-surface rounded-lg border border-carbon-border text-emerald-300 break-all select-all">
        {{ synthesizedCliPreview }}
      </div>
    </div>

    <!-- Action Buttons -->
    <div class="sticky bottom-0 -mx-1 px-3 py-2.5 bg-carbon-surface/95 backdrop-blur-md border-t border-carbon-border flex items-center justify-end gap-2 z-10 shadow-lg">
      <button
        type="button"
        @click="emit('cancel')"
        class="px-3 py-1.5 bg-carbon-elevated text-slate-300 rounded-lg text-xs cursor-pointer hover:bg-carbon-border transition"
      >
        Cancel
      </button>
      <button
        type="button"
        @click="handleSave()"
        class="px-4 py-1.5 bg-electric-blue hover:bg-electric-blue/90 text-white rounded-lg text-xs font-semibold cursor-pointer transition shadow-sm"
      >
        Save Configuration
      </button>
    </div>
  </div>
</template>
