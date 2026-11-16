<template>
  <div
    v-if="visible"
    class="fixed z-[9999] bg-[#111318] border border-[#262A36] rounded-xl shadow-2xl py-1.5 min-w-[14rem] font-sans text-xs text-slate-200 divide-y divide-[#262A36] select-none"
    :style="{ top: `${adjustedY}px`, left: `${adjustedX}px` }"
    @click.stop
  >
    <!-- Item Header -->
    <div class="px-3 py-1.5 bg-[#15181F] text-[10px] font-mono text-slate-400 uppercase tracking-wider truncate flex items-center justify-between gap-2">
      <span class="truncate font-semibold">{{ item?.Name }}</span>
      <span class="text-[9px] px-1.5 py-0.2 rounded bg-carbon-800 text-electric-400 border border-carbon-700">
        {{ remote }}
      </span>
    </div>

    <!-- File Inspection & Operations -->
    <div class="py-1">
      <button
        v-if="!item?.IsDir"
        @click="action('preview')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-800 hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-eye text-electric-400 text-xs"></i>
        <span>Preview File</span>
      </button>

      <button
        v-if="!item?.IsDir"
        @click="action('compare')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-800 hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-sliders-v text-indigo-400 text-xs"></i>
        <span>Toggle Compare Selection</span>
      </button>

      <button
        v-if="!item?.IsDir"
        @click="action('download')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-800 hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-download text-emerald-400 text-xs"></i>
        <span>Download File</span>
      </button>

      <button
        v-if="remote !== 'local'"
        @click="action('open_on_remote')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-800 hover:text-white flex items-center gap-2 cursor-pointer transition-colors text-electric-300"
      >
        <i class="pi pi-external-link text-electric-400 text-xs"></i>
        <span>Open on Remote</span>
      </button>
    </div>

    <!-- Management Actions -->
    <div class="py-1">
      <button
        @click="action('move')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-800 hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-file-edit text-amber-300 text-xs"></i>
        <span>Move / Rename</span>
      </button>

      <button
        @click="action('copy_path')"
        class="w-full text-left px-3 py-1.5 hover:bg-carbon-800 hover:text-white flex items-center gap-2 cursor-pointer transition-colors"
      >
        <i class="pi pi-copy text-slate-400 text-xs"></i>
        <span>Copy Path</span>
      </button>
    </div>

    <!-- Destructive Action -->
    <div class="py-1">
      <button
        @click="action('delete')"
        class="w-full text-left px-3 py-1.5 hover:bg-rose-950/60 text-rose-400 hover:text-rose-300 flex items-center gap-2 cursor-pointer transition-colors"
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
  (e: 'action', act: 'preview' | 'compare' | 'download' | 'open_on_remote' | 'move' | 'copy_path' | 'delete'): void
}>()

const adjustedX = computed(() => Math.min(props.x, window.innerWidth - 240))
const adjustedY = computed(() => Math.min(props.y, window.innerHeight - 280))

function action(act: 'preview' | 'compare' | 'download' | 'open_on_remote' | 'move' | 'copy_path' | 'delete') {
  emit('action', act)
  emit('close')
}

function handleGlobalClick() {
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
