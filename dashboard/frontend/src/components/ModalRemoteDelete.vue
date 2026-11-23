<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { useRemotesStore } from '../stores/remotes'

const props = defineProps<{
  visible: boolean
  remote: any | null
  descendants: any[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'deleted', name: string): void
}>()

const remotesStore = useRemotesStore()
const deleting = ref(false)
const error = ref<string | null>(null)

async function executeDelete() {
  if (!props.remote?.name) return
  deleting.value = true
  error.value = null
  try {
    await remotesStore.deleteRemote(props.remote.name)
    emit('deleted', props.remote.name)
    emit('update:visible', false)
  } catch (err: any) {
    error.value = err.message || 'Failed to delete storage remote'
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    :style="{ width: '34rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-rose-500/30 rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-4' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 shrink-0">
          <i class="pi pi-trash text-lg"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white">Delete Storage Remote</h3>
    <div v-if="remote" class="bg-carbon-bg border border-carbon-border rounded-xl p-3 space-y-1.5 font-mono text-xs">
      <div class="flex items-center justify-between text-slate-400">
        <span class="text-slate-500">Remote:</span>
        <span class="text-white font-bold truncate max-w-[16rem]">{{ remote.name }}</span>
      </div>
      <div class="flex items-center justify-between text-slate-400">
        <span class="text-slate-500">Type:</span>
        <span class="text-electric-blue font-semibold uppercase">{{ remote.type || 'unknown' }}</span>
      </div>
      <div class="flex items-center justify-between text-slate-400">
        <span class="text-slate-500">Target:</span>
        <span class="text-slate-300 truncate max-w-[16rem]">
          {{ remote.underlying_path || (remote.underlying_remote ? remote.underlying_remote + ':' : '') || remote.endpoint || remote.root || 'Configured endpoint' }}
        </span>
      </div>
    </div>

    <div class="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl space-y-1 text-xs">
      <p class="font-semibold text-rose-300 flex items-center gap-1.5">
        <i class="pi pi-exclamation-triangle text-rose-400"></i>
        <span>Permanent Removal Safeguard</span>
      </p>
      <p class="leading-relaxed text-[11px] text-slate-300">
        Permanently delete remote <span class="font-bold text-white font-mono">"{{ remote?.name }}"</span>?
        Active sync profiles referencing this remote will be impacted.
      </p>
    </div>

    <!-- Cascade Overlay Warning -->
    <div
      v-if="descendants && descendants.length > 0"
      class="p-3.5 bg-amber-950/40 border border-amber-800/60 rounded-xl space-y-2 text-xs font-mono"
    >
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-1.5 text-amber-300 font-bold uppercase tracking-wider text-[11px]">
          <i class="pi pi-exclamation-triangle text-amber-400 text-xs"></i>
          <span>Cascade Warning</span>
        </div>
        <span class="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-900/60 text-amber-300 border border-amber-700/60">
          {{ descendants.length }} dependent overlay{{ descendants.length > 1 ? 's' : '' }}
        </span>
      </div>
      <p class="text-[11px] text-amber-100/90 leading-relaxed font-sans">
        Deleting this remote will orphan or break these downstream overlays:
      </p>
      <div class="space-y-1.5 max-h-32 overflow-y-auto pr-1">
        <div
          v-for="d in descendants"
          :key="d.name"
          class="flex items-center justify-between bg-carbon-surface/90 border border-amber-800/40 px-2.5 py-1.5 rounded-lg text-xs"
        >
          <div class="flex items-center gap-2 min-w-0">
            <i v-if="(d.type || '').toLowerCase() === 'crypt'" class="pi pi-shield text-cyan-400 text-[10px]"></i>
            <i v-else-if="(d.type || '').toLowerCase() === 'hasher'" class="pi pi-bolt text-amber-400 text-[10px]"></i>
            <i v-else class="pi pi-link text-slate-400 text-[10px]"></i>
            <span class="text-white font-medium truncate">{{ d.name }}</span>
          </div>
          <span class="text-[10px] px-1.5 py-0.5 rounded bg-carbon-canvas border border-carbon-border text-slate-300 font-mono uppercase">
            {{ d.type || 'overlay' }}
          </span>
        </div>
      </div>
    </div>

    <!-- Crypt Key Deletion Warning -->
    <div
      v-if="remote && (remote.type || '').toLowerCase() === 'crypt'"
      class="p-3 bg-cyan-950/30 border border-cyan-800/50 rounded-xl space-y-1.5 text-xs font-mono"
    >
      <div class="flex items-center gap-1.5 text-cyan-300 font-semibold text-[11px]">
        <i class="pi pi-shield text-cyan-400 text-xs"></i>
        <span>Decryption Key Removal Safeguard</span>
      </div>
      <p class="leading-relaxed text-[11px] text-slate-300 font-sans">
        Deleting this crypt remote removes local decryption keys from <code class="text-cyan-300 font-mono">rclone.conf</code>.
        Encrypted cipher files in cloud storage remain unreadable ciphertext unless decrypted prior to deletion.
      </p>
    </div>

    <div v-if="error" class="p-2.5 bg-rose-900/40 border border-rose-700 rounded-lg text-xs text-rose-200">
      {{ error }}
    </div>

    <div class="flex items-center justify-end gap-2 pt-2 border-t border-carbon-border">
      <button
        type="button"
        @click="emit('update:visible', false)"
        class="px-3 py-1.5 bg-carbon-elevated hover:bg-carbon-border text-slate-300 rounded-lg text-xs font-medium transition cursor-pointer"
      >
        Cancel
      </button>
      <button
        type="button"
        @click="executeDelete()"
        :disabled="deleting"
        class="px-3 py-1.5 bg-rose-600 hover:bg-rose-500 text-white rounded-lg text-xs font-semibold transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
      >
        <i v-if="deleting" class="pi pi-spin pi-spinner text-xs"></i>
        <span>{{ deleting ? 'Deleting...' : 'Permanently Delete Remote' }}</span>
      </button>
    </div>
          <div class="text-[11px] text-slate-400">Irreversible configuration removal</div>
        </div>
      </div>
    </template>
  </Dialog>
</template>