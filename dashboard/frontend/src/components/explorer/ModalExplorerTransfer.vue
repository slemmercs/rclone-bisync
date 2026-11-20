<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    :header="modalTitle"
    :style="{ width: '36rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between text-white font-bold' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <div class="space-y-4 text-xs font-mono">
      <div>
        <label class="block text-slate-400 font-sans font-semibold mb-1.5 uppercase text-[10px] tracking-wider">Transfer Action</label>
        <div class="grid grid-cols-2 gap-2">
          <button
            type="button"
            @click="transferAction = 'copy'"
            class="p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2"
            :class="transferAction === 'copy' ? 'bg-indigo-950/80 border-indigo-500 text-white' : 'bg-carbon-elevated border-carbon-border text-slate-400 hover:text-slate-200'"
          >
            <i class="pi pi-copy text-indigo-400 text-sm"></i>
            <div>
              <div class="font-bold text-xs">Copy Item</div>
              <div class="text-[10px] text-slate-400 font-sans">Duplicate to destination</div>
            </div>
          </button>
          <button
            type="button"
            @click="transferAction = 'move'"
            class="p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2"
            :class="transferAction === 'move' ? 'bg-amber-950/80 border-amber-500 text-white' : 'bg-carbon-elevated border-carbon-border text-slate-400 hover:text-slate-200'"
          >
            <i class="pi pi-arrows-alt text-amber-400 text-sm"></i>
            <div>
              <div class="font-bold text-xs">Move Item</div>
              <div class="text-[10px] text-slate-400 font-sans">Transfer & remove source</div>
            </div>
          </button>
        </div>
      </div>

      <div class="space-y-1">
        <label class="block text-slate-400 font-sans font-semibold text-[10px] uppercase tracking-wider">Source</label>
        <div class="px-3 py-2 bg-carbon-elevated rounded-xl border border-carbon-border text-slate-300 break-all flex items-center gap-2">
          <i :class="isDir ? 'pi pi-folder text-amber-400' : 'pi pi-file text-slate-400'" class="text-xs shrink-0"></i>
          <span class="text-electric-blue font-bold uppercase shrink-0">{{ sourceRemote }}:</span>
          <span class="truncate">{{ sourcePath }}</span>
        </div>
      </div>

      <div class="space-y-1">
        <label class="block text-slate-400 font-sans font-semibold text-[10px] uppercase tracking-wider">Destination</label>
        <div class="flex items-center gap-2">
          <span class="px-2.5 py-2 bg-carbon-canvas border border-carbon-border rounded-xl font-bold uppercase text-emerald-400 text-xs shrink-0">
            {{ destRemote }}:
          </span>
          <input
            v-model="targetPath"
            type="text"
            class="flex-1 bg-carbon-elevated border border-carbon-border rounded-xl px-3 py-2 text-white font-mono focus:outline-none focus:border-electric-blue text-xs"
            placeholder="Relative destination path..."
            @keydown.enter="handleConfirm"
          />
        </div>
      </div>

      <div v-if="error" class="p-2.5 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl text-xs flex items-center gap-2">
        <i class="pi pi-exclamation-triangle text-rose-400 shrink-0"></i>
        <span>{{ error }}</span>
      </div>
    </div>

    <div class="flex items-center justify-end gap-2 border-t border-carbon-border pt-3 mt-4">
      <button
        type="button"
        @click="emit('update:visible', false)"
        class="px-3 py-1.5 bg-carbon-elevated text-slate-300 hover:bg-slate-800 rounded-xl text-xs cursor-pointer"
      >
        Cancel
      </button>
      <button
        type="button"
        @click="handleConfirm"
        :disabled="loading || !targetPath.trim()"
        class="px-4 py-1.5 rounded-xl text-xs font-semibold cursor-pointer disabled:opacity-50 flex items-center gap-1.5 shadow-lg"
        :class="transferAction === 'copy' ? 'bg-indigo-600 hover:bg-indigo-500 text-white' : 'bg-amber-600 hover:bg-amber-500 text-white'"
      >
        <i v-if="loading" class="pi pi-spin pi-spinner text-xs"></i>
        <i v-else :class="transferAction === 'copy' ? 'pi pi-copy' : 'pi pi-arrow-right'" class="text-xs"></i>
        <span>{{ loading ? 'Transferring...' : (transferAction === 'copy' ? 'Execute Copy' : 'Execute Move') }}</span>
      </button>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'

const props = defineProps<{
  visible: boolean
  sourceRemote: string
  sourcePath: string
  destRemote: string
  destPath: string
  itemName: string
  isDir: boolean
  loading?: boolean
  error?: string | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'confirm', action: 'copy' | 'move', dstPath: string): void
}>()

const transferAction = ref<'copy' | 'move'>('copy')
const targetPath = ref('')

const modalTitle = computed(() => {
  const op = transferAction.value === 'copy' ? 'Copy' : 'Move'
  return `${op} ${props.isDir ? 'Directory' : 'File'} between Endpoints`
})

watch(
  () => props.visible,
  (val) => {
    if (val) {
      targetPath.value = props.destPath
        ? `${props.destPath.replace(/\/+$/, '')}/${props.itemName}`
        : props.itemName
      transferAction.value = 'copy'
    }
  },
  { immediate: true }
)

function handleConfirm() {
  if (!targetPath.value.trim() || props.loading) return
  emit('confirm', transferAction.value, targetPath.value.trim())
}
</script>
