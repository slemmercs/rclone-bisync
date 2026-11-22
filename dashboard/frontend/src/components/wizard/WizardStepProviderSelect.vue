<template>
  <div class="space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-1.5 text-xs font-mono">
        <button
          v-for="cat in (['popular', 'cloud', 'object', 'protocol', 'all'] as const)"
          :key="cat"
          type="button"
          @click="category = cat"
          :class="category === cat ? 'bg-electric-blue text-white font-bold' : 'bg-carbon-elevated text-slate-300 hover:text-white'"
          class="px-2.5 py-1 rounded-lg transition-colors cursor-pointer capitalize"
        >
          {{ cat === 'object' ? 'Object (S3)' : cat }}
        </button>
      </div>
      <div class="relative w-full sm:w-60">
        <input
          type="text"
          v-model="search"
          placeholder="Search providers..."
          class="w-full bg-carbon-surface border border-carbon-border rounded-lg pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-electric-blue font-mono"
        />
        <i class="pi pi-search text-slate-500 absolute left-2.5 top-2.5 text-xs pointer-events-none"></i>
      </div>
    </div>

    <div v-if="loading" class="py-16 text-center text-slate-400 space-y-2">
      <i class="pi pi-spin pi-spinner text-xl text-electric-blue"></i>
      <div class="text-xs font-mono">Fetching provider catalog from Rclone...</div>
    </div>

    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <div
        v-for="p in filteredProviders"
        :key="p.name"
        @click="emit('select', p)"
        class="p-3.5 rounded-xl border bg-carbon-surface/60 hover:bg-carbon-elevated/80 transition-all cursor-pointer flex flex-col justify-between group space-y-2"
        :class="selected?.name === p.name ? 'border-electric-blue ring-1 ring-electric-blue' : 'border-carbon-border hover:border-carbon-border/80'"
      >
        <div class="space-y-1.5">
          <div class="flex items-center justify-between gap-2">
            <span class="font-bold text-sm text-white group-hover:text-electric-300 transition-colors">{{ p.name }}</span>
            <div class="flex items-center gap-1">
              <span v-if="p.category === 'popular'" class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-electric-blue/10 text-electric-blue border border-electric-blue/20">Popular</span>
              <span v-if="isOAuthProvider(p)" class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">OAuth</span>
            </div>
          </div>
          <p class="text-xs text-slate-400 line-clamp-2">{{ p.description || p.name }}</p>
        </div>
        <div class="flex items-center justify-between text-[11px] text-slate-500 font-mono pt-2 border-t border-carbon-border">
          <span>{{ p.prefix || p.name }}</span>
          <span class="text-electric-blue group-hover:translate-x-0.5 transition-transform">Configure →</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { ProviderCatalogItem } from '../../types'
import { isOAuthProvider } from '../../types'

const props = defineProps<{
  providers: ProviderCatalogItem[]
  loading?: boolean
  selected: ProviderCatalogItem | null
}>()

const emit = defineEmits<{
  (e: 'select', provider: ProviderCatalogItem): void
}>()

const category = ref<'popular' | 'cloud' | 'object' | 'protocol' | 'all'>('popular')
const search = ref<string>('')

const filteredProviders = computed(() => {
  let list = props.providers || []
  if (category.value === 'popular') {
    list = list.filter(p => p.category === 'popular' || ['drive', 'onedrive', 's3', 'b2', 'dropbox'].includes(p.name))
  } else if (category.value !== 'all') {
    list = list.filter(p => p.category === category.value)
  }

  const q = search.value.toLowerCase().trim()
  if (q) {
    list = list.filter(p => (p.name || '').toLowerCase().includes(q) || (p.description || '').toLowerCase().includes(q))
  }
  return list
})
</script>
