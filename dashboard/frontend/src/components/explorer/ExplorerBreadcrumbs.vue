<template>
  <div class="mb-2 bg-carbon-950/80 px-3 py-1.5 rounded-xl border border-carbon-800 flex items-center justify-between gap-2 text-xs font-mono text-slate-300 min-w-0 shrink-0">
    <div class="flex items-center gap-1 min-w-0 overflow-x-auto py-0.5 no-scrollbar flex-1" :title="displayPath">
      <i class="pi pi-folder text-slate-500 text-xs shrink-0 mr-1"></i>
      <template v-for="(crumb, idx) in breadcrumbs" :key="crumb.path + crumb.name">
        <div class="flex items-center shrink-0 min-w-0">
          <span v-if="idx > 0" class="text-carbon-600 px-1 select-none">/</span>
          <button
            type="button"
            @click="$emit('navigate', crumb)"
            class="transition-colors truncate max-w-[140px] sm:max-w-[200px] cursor-pointer"
            :class="[
              idx === breadcrumbs.length - 1 ? 'text-electric-400 font-bold' : 'text-slate-400 hover:text-white hover:underline',
              crumb.isEllipsis ? 'text-slate-500 hover:text-slate-300 font-bold px-0.5' : '',
            ]"
            :title="crumb.title || crumb.name"
          >
            {{ crumb.name }}
          </button>
        </div>
      </template>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <span class="text-[10px] text-carbon-500 font-sans">{{ itemCount }} item(s)</span>
      <button
        type="button"
        @click="$emit('copy')"
        class="p-1 hover:bg-carbon-800 text-slate-400 hover:text-white rounded transition-colors cursor-pointer"
        :class="copied ? 'text-emerald-400' : ''"
        :title="copied ? 'Copied!' : 'Copy Path'"
      >
        <i :class="copied ? 'pi pi-check text-xs' : 'pi pi-copy text-xs'"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BreadcrumbItem } from '../../types/explorer'

defineProps<{
  displayPath: string
  breadcrumbs: BreadcrumbItem[]
  itemCount: number
  copied: boolean
}>()

defineEmits<{
  (e: 'navigate', crumb: BreadcrumbItem): void
  (e: 'copy'): void
}>()
</script>
