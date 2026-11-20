<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    :header="preview?.name || 'File Preview'"
    :style="{ width: '56rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between text-white font-bold' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <!-- File Metadata Bar -->
    <div class="flex items-center justify-between bg-carbon-elevated p-2.5 rounded-xl border border-carbon-border text-xs font-mono">
      <div class="flex items-center gap-2 truncate">
        <span class="px-2 py-0.5 rounded bg-carbon-surface text-electric-blue border border-carbon-border uppercase font-semibold text-[10px]">
          {{ preview?.remote || 'local' }}
        </span>
        <span class="text-slate-300 truncate">{{ preview?.path }}</span>
      </div>
      <div class="flex items-center gap-2 text-slate-400 shrink-0">
        <span>{{ formatBytes(preview?.size) }}</span>
        <span>•</span>
        <span class="text-slate-500">{{ preview?.mime_type || 'text/plain' }}</span>
      </div>
    </div>

    <!-- Truncation / Duplicate Notices -->
    <div v-if="preview?.truncated" class="p-2.5 bg-amber-500/10 border border-amber-500/20 rounded-xl text-xs text-amber-300 flex items-center gap-2">
      <i class="pi pi-exclamation-triangle text-amber-400"></i>
      <span>Preview truncated to 2 MB limit. Download file for complete content.</span>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="py-16 text-center text-slate-400 space-y-3">
      <i class="pi pi-spin pi-spinner text-2xl text-electric-blue"></i>
      <div class="text-xs font-mono">Loading preview payload...</div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="p-4 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl text-xs font-mono">
      {{ error }}
    </div>

    <!-- Image View -->
    <div v-else-if="preview?.data_url" class="flex items-center justify-center p-3 bg-carbon-canvas rounded-xl border border-carbon-border">
      <img :src="preview.data_url" :alt="preview.name" class="max-h-[60vh] max-w-full rounded-lg object-contain shadow-md" />
    </div>

    <!-- PDF Viewer via Iframe -->
    <div v-else-if="preview?.is_pdf" class="w-full h-[65vh] bg-carbon-canvas rounded-xl border border-carbon-border overflow-hidden">
      <iframe :src="downloadUrl + '&inline=true'" class="w-full h-full border-0 rounded-xl" title="PDF Preview"></iframe>
    </div>

    <!-- Binary File Warning -->
    <div v-else-if="preview?.is_binary" class="py-12 flex flex-col items-center justify-center text-center gap-3">
      <div class="w-12 h-12 rounded-full bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-400">
        <i class="pi pi-ban text-xl"></i>
      </div>
      <div class="space-y-1">
        <h4 class="text-sm font-semibold text-white">Preview Not Allowed</h4>
        <p class="text-xs text-slate-400 max-w-sm">
          {{ preview.error || 'Preview is not allowed for binary files. Please download the file to inspect its contents.' }}
        </p>
      </div>
    </div>

    <!-- Plaintext / Code View -->
    <div v-else class="max-h-[60vh] overflow-auto p-4 bg-carbon-canvas rounded-xl border border-carbon-border font-mono text-xs text-slate-200 leading-relaxed whitespace-pre select-text">
      <code>{{ preview?.content || '(empty file)' }}</code>
    </div>

    <!-- Modal Footer -->
    <div class="flex items-center justify-between border-t border-carbon-border pt-3">
      <span class="text-[11px] text-slate-500 font-mono">Read-only jail confinement</span>
      <div class="flex items-center gap-2">
        <a
          :href="downloadUrl"
          download
          class="px-3 py-1.5 bg-carbon-elevated hover:bg-slate-800 text-electric-blue border border-carbon-border rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 cursor-pointer"
        >
          <i class="pi pi-download text-xs"></i>
          <span>Download</span>
        </a>
        <button
          @click="emit('update:visible', false)"
          class="px-3 py-1.5 bg-carbon-elevated text-slate-300 rounded-xl text-xs cursor-pointer hover:bg-slate-800"
        >
          Close
        </button>
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Dialog from 'primevue/dialog'
import type { FilePreviewResponse } from '../types/explorer'

const props = defineProps<{
  visible: boolean
  preview: FilePreviewResponse | null
  loading?: boolean
  error?: string | null
  profileId?: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

const downloadUrl = computed(() => {
  if (!props.preview) return '#'
  const params = new URLSearchParams({
    remote: props.preview.remote || 'local',
    path: props.preview.path || '',
    ...(props.profileId ? { profile_id: props.profileId } : {}),
  })
  return `/api/explorer/download?${params.toString()}`
})

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null || bytes < 0) return '--'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}
</script>
