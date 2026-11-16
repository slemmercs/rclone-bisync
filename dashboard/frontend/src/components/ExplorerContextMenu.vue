<template>
  <div
    v-if="visible"
    class="fixed z-50 bg-carbon-surface border border-carbon-border rounded-xl shadow-2xl py-1.5 min-w-[13rem] font-sans text-xs text-slate-200 divide-y divide-carbon-border/50 animate-in fade-in zoom-in-95 duration-100"
    :style="{ top: `${adjustedY}px`, left: `${adjustedX}px` }"
    @click.stop
  >
    <div class="px-3 py-1 text-[10px] font-mono text-slate-500 uppercase tracking-wider truncate max-w-[15rem]">
      {{ item?.Name }}
    </div>

    <div class="py-1">
      <button
        v-if="!item?.IsDir"
        @click="action('preview')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-elevated hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-eye text-electric-blue text-xs"></i>
        <span>Preview File</span>
      </button>

      <button
        v-if="!item?.IsDir"
        @click="action('compare')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-elevated hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-sliders-v text-indigo-400 text-xs"></i>
        <span>Compare Hashes</span>
      </button>

      <button
        v-if="!item?.IsDir"
        @click="action('download')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-elevated hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-download text-emerald-400 text-xs"></i>
        <span>Download File</span>
      </button>
    </div>

    <div class="py-1">
      <button
        @click="action('move')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-elevated hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-file-edit text-amber-300 text-xs"></i>
        <span>Move / Rename</span>
      </button>

      <button
        @click="action('copy_path')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-elevated hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-copy text-slate-400 text-xs"></i>
        <span>Copy Path</span>
      </button>
    </div>

    <div class="py-1">
      <button
        @click="action('delete')"
        class="w-full text-left px-3 py-1.5 hover:bg-rose-500/20 text-rose-400 hover:text-rose-300 flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-trash text-xs"></i>
        <span>{{ item?.IsDir ? 'Purge Directory' : 'Delete Item' }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import type { BrowseItem } from '../types/models'

const props = defineProps<{
  visible: boolean
  x: number
  y: number
  item: BrowseItem | null
  remote: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'action', act: 'preview' | 'compare' | 'download' | 'move' | 'copy_path' | 'delete'): void
}>()

const adjustedX = computed(() => Math.min(props.x, window.innerWidth - 220))
const adjustedY = computed(() => Math.min(props.y, window.innerHeight - 250))

function action(act: 'preview' | 'compare' | 'download' | 'move' | 'copy_path' | 'delete') {
  emit('action', act)
  emit('close')
}

function handleGlobalClick(e: MouseEvent) {
  if (props.visible) {
    emit('close')
  }
}

function handleKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.visible) {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('click', handleGlobalClick)
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('click', handleGlobalClick)
  window.removeEventListener('keydown', handleKeyDown)
})
</script>
