<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    :header="isDir ? 'Move or Rename Directory' : 'Move or Rename File'"
    :style="{ width: '34rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between text-white font-bold' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <div class="space-y-3 text-xs">
      <div>
        <label class="block text-slate-400 font-semibold mb-1">Source Item</label>
        <div class="px-3 py-2 bg-carbon-elevated rounded-xl border border-carbon-border font-mono text-slate-300 break-all">
          <span class="text-electric-blue font-bold uppercase">{{ remote }}</span>:{{ srcPath }}
        </div>
      </div>

      <div>
        <label class="block text-slate-400 font-semibold mb-1">Destination Relative Path</label>
        <input
          v-model="dstPath"
          type="text"
          class="w-full bg-carbon-elevated border border-carbon-border rounded-xl p-2.5 text-white font-mono focus:outline-none focus:border-electric-blue"
          @keydown.enter="handleMove"
        />
        <p class="text-[10px] text-slate-500 mt-1">Relative to the jail root of the active storage endpoint.</p>
      </div>

      <div v-if="error" class="p-2.5 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl font-mono text-xs">
        {{ error }}
      </div>
    </div>

    <div class="flex items-center justify-end gap-2 border-t border-carbon-border pt-3">
      <button
        @click="emit('update:visible', false)"
        class="px-3 py-1.5 bg-carbon-elevated text-slate-300 hover:bg-slate-800 rounded-xl text-xs cursor-pointer"
      >
        Cancel
      </button>
      <button
        @click="handleMove"
        :disabled="loading || !dstPath.trim()"
        class="px-4 py-1.5 bg-electric-blue hover:bg-sky-400 text-white rounded-xl text-xs font-semibold cursor-pointer disabled:opacity-50 flex items-center gap-1.5"
      >
        <i v-if="loading" class="pi pi-spin pi-spinner text-xs"></i>
        <span>{{ loading ? 'Moving...' : 'Move / Rename' }}</span>
      </button>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'

const props = defineProps<{
  visible: boolean
  remote: string
  srcPath: string
  isDir: boolean
  loading?: boolean
  error?: string | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'move', dstPath: string): void
}>()

const dstPath = ref('')

watch(
  () => props.srcPath,
  (val) => {
    dstPath.value = val || ''
  },
  { immediate: true }
)

function handleMove() {
  if (!dstPath.value.trim() || props.loading) return
  emit('move', dstPath.value.trim())
}
</script>
