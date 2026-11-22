<template>
  <Dialog
    :visible="remotesStore.hasherModal.show"
    @update:visible="(val) => remotesStore.hasherModal.show = val"
    modal
    :style="{ width: '42rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div class="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400 flex items-center justify-center">
          <i class="pi pi-bolt text-base"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white flex items-center gap-2">
            <span>Hasher Acceleration Console</span>
            <span v-if="remotesStore.hasherModal.remote" class="text-[10px] font-mono px-2 py-0.5 rounded bg-amber-950/60 text-amber-300 border border-amber-800/60 font-semibold">
              {{ remotesStore.hasherModal.remote }}
            </span>
          </h3>
          <p class="text-[11px] text-slate-400 font-sans">
            Rclone Hasher backend checksum caching, deduplication acceleration, and DB maintenance.
          </p>
        </div>
      </div>
    </template>

    <!-- Mode Selector Tabs -->
    <div class="flex items-center bg-carbon-canvas p-1 rounded-xl border border-carbon-border text-xs font-mono">
      <button
        type="button"
        @click="activeTab = 'commands'"
        :class="activeTab === 'commands' ? 'bg-amber-500 text-slate-950 font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
        class="flex-1 py-1.5 rounded-lg text-center transition-all cursor-pointer flex items-center justify-center gap-1.5"
      >
        <i class="pi pi-terminal text-xs"></i>
        <span>Maintenance Commands</span>
      </button>
      <button
        type="button"
        @click="activeTab = 'wrap'"
        :class="activeTab === 'wrap' ? 'bg-amber-500 text-slate-950 font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
        class="flex-1 py-1.5 rounded-lg text-center transition-all cursor-pointer flex items-center justify-center gap-1.5"
      >
        <i class="pi pi-plus-circle text-xs"></i>
        <span>Wrap Remote in Hasher</span>
      </button>
    </div>

    <!-- TAB 1: Maintenance Commands -->
    <div v-if="activeTab === 'commands'" class="space-y-4 font-mono text-xs">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">Target Hasher Remote</label>
          <select
            v-if="hasherRemotes.length > 0"
            v-model="cmdRemote"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400 cursor-pointer"
          >
            <option v-for="r in hasherRemotes" :key="r.name" :value="r.name">{{ r.name }}</option>
          </select>
          <input
            v-else
            type="text"
            v-model="cmdRemote"
            placeholder="e.g. drive_hasher"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400"
          />
        </div>

        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">Command</label>
          <select
            v-model="selectedCmd"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400 cursor-pointer"
          >
            <option value="dump">dump — Export cached checksums</option>
            <option value="fulldump">fulldump — Full export including untracked</option>
            <option value="drop">drop — Evict / purge checksum records</option>
            <option value="import">import — Import checksums from file</option>
            <option value="stickyimport">stickyimport — Sticky import records</option>
          </select>
        </div>

        <div class="sm:col-span-2">
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">Command Arguments / Path (Optional)</label>
          <input
            type="text"
            v-model="cmdArgs"
            placeholder="e.g. subfolder/file.txt (leave empty for full remote scope)"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400"
          />
        </div>
      </div>
      <!-- Action Button -->
      <div class="flex items-center justify-between pt-1">
        <span class="text-[11px] text-slate-500 font-sans">
          Executes Rclone <code>backend command hasher</code> over RC API.
        </span>
        <button
          type="button"
          @click="runCommand"
          :disabled="!cmdRemote || executingCmd"
          class="px-4 py-1.5 bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold rounded-lg text-xs font-mono transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-40"
        >
          <i class="pi" :class="executingCmd ? 'pi-spin pi-spinner' : 'pi-play'"></i>
          <span>{{ executingCmd ? 'Running...' : 'Execute Command' }}</span>
        </button>
      </div>

      <!-- Command Output Console -->
      <div v-if="cmdOutput !== null || cmdError !== null" class="space-y-1.5">
        <div class="flex items-center justify-between text-[11px] text-slate-400">
          <span class="flex items-center gap-1.5">
            <i class="pi pi-code text-amber-400"></i>
            <span>Execution Output</span>
            <span v-if="cmdError" class="text-rose-400 font-bold">[FAILED]</span>
            <span v-else class="text-emerald-400 font-bold">[SUCCESS]</span>
          </span>
          <button
            type="button"
            @click="copyOutput"
            class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans flex items-center gap-1"
          >
            <i class="pi pi-copy text-[10px]"></i>
            <span>{{ copied ? 'Copied!' : 'Copy Output' }}</span>
          </button>
        </div>

        <pre
          class="w-full bg-carbon-canvas border border-carbon-border rounded-xl p-3 text-[11px] font-mono overflow-x-auto max-h-56 leading-relaxed select-all"
          :class="cmdError ? 'text-rose-300 border-rose-500/30' : 'text-slate-200'"
        >{{ cmdError || cmdOutput }}</pre>
      </div>
    </div>

    <!-- TAB 2: Wrap Remote in Hasher -->
    <div v-else class="space-y-4 font-mono text-xs">
      <div class="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-300 text-[11px] font-sans flex items-center gap-2">
        <i class="pi pi-info-circle text-amber-400 shrink-0 text-sm"></i>
        <span>
          A Hasher overlay calculates and caches cryptographic hashes locally, dramatically speeding up bisync checks on non-hash remotes.
        </span>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <!-- Base Remote -->
        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">Underlying Remote *</label>
          <select
            v-if="availableBaseRemotes.length > 0"
            v-model="wrapBaseRemote"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400 cursor-pointer"
          >
            <option v-for="r in availableBaseRemotes" :key="r.name" :value="r.name">{{ r.name }} ({{ r.type }})</option>
          </select>
          <input
            v-else
            type="text"
            v-model="wrapBaseRemote"
            placeholder="e.g. mydrive"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400"
          />
        </div>

        <!-- Hasher Remote Name -->
        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">New Hasher Remote Name *</label>
          <input
            type="text"
            v-model="wrapHasherName"
            placeholder="e.g. mydrive_hasher"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400"
          />
        </div>

        <!-- Max Age -->
        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">Max Age (Cache Expiration)</label>
          <input
            type="text"
            v-model="wrapMaxAge"
            placeholder="off (forever) or 24h, 7d"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400"
          />
        </div>
      </div>
        <!-- Hashes Selection -->
        <div class="sm:col-span-2">
          <label class="block text-slate-300 mb-1.5 text-[11px] font-semibold">Hash Algorithms to Cache</label>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="h in availableHashes"
              :key="h.value"
              class="px-2.5 py-1 rounded-lg border text-[11px] font-mono cursor-pointer flex items-center gap-1.5 select-none transition-colors"
              :class="
                selectedHashes.includes(h.value)
                  ? 'bg-amber-500/20 text-amber-300 border-amber-500/40 font-bold'
                  : 'bg-carbon-canvas text-slate-400 border-carbon-border hover:border-slate-600'
              "
            >
              <input
                type="checkbox"
                :value="h.value"
                v-model="selectedHashes"
                class="hidden"
              />
              <i class="pi" :class="selectedHashes.includes(h.value) ? 'pi-check text-amber-400 text-[10px]' : 'pi-circle text-[10px] opacity-40'"></i>
              <span>{{ h.label }}</span>
            </label>
          </div>
        </div>

      <div v-if="wrapError" class="p-3 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl text-xs">
        {{ wrapError }}
      </div>

      <div v-if="wrapSuccess" class="p-3 bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 rounded-xl text-xs">
        {{ wrapSuccess }}
      </div>

      <div class="flex items-center justify-end gap-2 pt-2 border-t border-carbon-border/60">
        <button
          type="button"
          @click="remotesStore.closeHasherManager()"
          class="px-3.5 py-1.5 rounded-lg bg-carbon-canvas hover:bg-slate-800 text-slate-300 text-xs font-mono border border-carbon-border cursor-pointer transition-colors"
        >
          Cancel
        </button>
        <button
          type="button"
          @click="handleWrap"
          :disabled="!wrapBaseRemote || !wrapHasherName || selectedHashes.length === 0 || wrapping"
          class="px-4 py-1.5 bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold rounded-lg text-xs font-mono transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-40"
        >
          <i class="pi" :class="wrapping ? 'pi-spin pi-spinner' : 'pi-bolt'"></i>
          <span>{{ wrapping ? 'Provisioning...' : 'Provision Hasher Remote' }}</span>
        </button>
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useRemotesStore } from '../stores/remotes'
import type { HasherWrapRequest } from '../types'

const remotesStore = useRemotesStore()

const activeTab = ref<'commands' | 'wrap'>('commands')

// Commands tab state
const cmdRemote = ref<string>('')
const selectedCmd = ref<string>('dump')
const cmdArgs = ref<string>('')
const executingCmd = ref<boolean>(false)
const cmdOutput = ref<string | null>(null)
const cmdError = ref<string | null>(null)
const copied = ref<boolean>(false)

// Wrap tab state
const wrapBaseRemote = ref<string>('')
const wrapHasherName = ref<string>('')
const wrapMaxAge = ref<string>('off')

interface HashOption {
  label: string
  value: string
}

const availableHashes: HashOption[] = [
  { label: 'MD5', value: 'md5' },
  { label: 'SHA-1', value: 'sha1' },
  { label: 'DropboxHash', value: 'dropbox' },
  { label: 'QuickXorHash', value: 'quickxor' },
  { label: 'Whirlpool', value: 'whirlpool' },
  { label: 'CRC-32', value: 'crc32' },
]
const selectedHashes = ref<string[]>(['md5', 'sha1'])
const wrapping = ref<boolean>(false)
const wrapError = ref<string | null>(null)
const wrapSuccess = ref<string | null>(null)

const hasherRemotes = computed(() => {
  return (remotesStore.remotes || []).filter(r => (r.type || '').toLowerCase() === 'hasher')
})

const availableBaseRemotes = computed(() => {
  return (remotesStore.remotes || []).filter(r => (r.type || '').toLowerCase() !== 'hasher')
})

watch(
  () => remotesStore.hasherModal.show,
  (show) => {
    if (!show) return
    activeTab.value = remotesStore.hasherModal.mode === 'wrap' ? 'wrap' : 'commands'
    const target = remotesStore.hasherModal.remote || ''

    if (activeTab.value === 'commands') {
      cmdRemote.value = target || (hasherRemotes.value[0]?.name || '')
      cmdOutput.value = null
      cmdError.value = null
    } else {
      wrapBaseRemote.value = target || (availableBaseRemotes.value[0]?.name || '')
      wrapHasherName.value = wrapBaseRemote.value ? `${wrapBaseRemote.value}_hasher` : ''
      wrapError.value = null
      wrapSuccess.value = null
    }
  }
)

watch(wrapBaseRemote, (newBase) => {
  if (newBase && (!wrapHasherName.value || wrapHasherName.value.endsWith('_hasher'))) {
    wrapHasherName.value = `${newBase}_hasher`
  }
})

async function runCommand() {
  if (!cmdRemote.value) return
  executingCmd.value = true
  cmdOutput.value = null
  cmdError.value = null

  try {
    const args = cmdArgs.value.trim() ? cmdArgs.value.trim().split(/\s+/) : []
    const res = await remotesStore.executeHasherCommand(cmdRemote.value, selectedCmd.value, args)
    if (res.error) {
      cmdError.value = res.error
    } else {
      cmdOutput.value = typeof res.result === 'string'
        ? res.result
        : JSON.stringify(res.result || res.output || res, null, 2)
    }
  } catch (err: any) {
    cmdError.value = err.message || 'Hasher command failed'
  } finally {
    executingCmd.value = false
  }
}

async function handleWrap() {
  if (!wrapBaseRemote.value || !wrapHasherName.value || selectedHashes.value.length === 0) return
  wrapping.value = true
  wrapError.value = null
  wrapSuccess.value = null

  const req: HasherWrapRequest = {
    name: wrapHasherName.value.trim(),
    remote: wrapBaseRemote.value.trim(),
    hashes: selectedHashes.value,
    max_age: wrapMaxAge.value.trim() || 'off',
  }

  try {
    await remotesStore.wrapHasher(req)
    wrapSuccess.value = `Successfully created Hasher accelerated remote "${req.name}" wrapping "${req.remote}"!`
    setTimeout(() => {
      remotesStore.closeHasherManager()
    }, 1500)
  } catch (err: any) {
    wrapError.value = err.message || 'Failed to wrap remote in Hasher'
  } finally {
    wrapping.value = false
  }
}

function copyOutput() {
  const text = cmdError.value || cmdOutput.value || ''
  if (!text) return
  navigator.clipboard.writeText(text)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}
</script>


