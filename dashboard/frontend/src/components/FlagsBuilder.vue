<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import {
  RCLONE_FLAGS_CATALOG,
  validateCustomFlagsString,
  getFlagCatalogItem,
  getFlagDomain,
  type FlagCatalogItem,
  type FlagDomain,
} from '../data/flags-catalog'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    remoteType?: string
    scopeFilter?: 'all' | 'engine' | 'profile'
  }>(),
  {
    modelValue: '',
    remoteType: 'all',
    scopeFilter: 'all',
  }
)
const emit = defineEmits<{ (e: 'update:modelValue', val: string): void }>()

interface FlagRow {
  id: string
  flag: string
  value: string
  isBool: boolean
  isCustom: boolean
  placeholder?: string
}

type FilterTab = 'all' | 'bisync' | 'rclone' | 'custom'

const rows = ref<FlagRow[]>([])
const activeFilter = ref<FilterTab>('all')
const validationError = computed(() => validateCustomFlagsString(props.modelValue || ''))

const availableCatalog = computed<FlagCatalogItem[]>(() => {
  const t = props.remoteType?.toLowerCase() || 'all'
  return RCLONE_FLAGS_CATALOG.filter((c) => {
    const backendMatch = !c.backends || c.backends.includes('all') || c.backends.includes(t)
    const scopeMatch = props.scopeFilter === 'all' || c.scope === props.scopeFilter
    return backendMatch && scopeMatch
  })
})

const bisyncCatalog = computed(() => availableCatalog.value.filter((c) => c.domain === 'bisync'))
const rcloneCatalog = computed(() => availableCatalog.value.filter((c) => c.domain === 'rclone'))

function getRowDomain(row: FlagRow): FlagDomain {
  if (row.isCustom) return 'custom'
  return getFlagDomain(row.flag)
}

function parseFlags(raw?: string): FlagRow[] {
  if (!raw?.trim()) return []
  const tokens = raw.trim().match(/[^\s"']+|"([^"]*)"|'([^']*)'/g) || []
  const parsed: FlagRow[] = []
  let i = 0
  while (i < tokens.length) {
    const clean = tokens[i].trim().replace(/^["']|["']$/g, '')
    let fn = clean, val = '', inline = false
    if (clean.includes('=')) {
      const p = clean.split('=')
      fn = p[0]; val = p.slice(1).join('='); inline = true
    }
    const cat = RCLONE_FLAGS_CATALOG.find((c) => c.flag === fn)
    const isBool = cat ? cat.isBool : false
    if (!isBool && !inline && i + 1 < tokens.length && !tokens[i + 1].startsWith('-')) {
      i++; val = tokens[i].replace(/^["']|["']$/g, '')
    }
    parsed.push({
      id: 'f_' + Math.random().toString(36).slice(2, 8),
      flag: fn, value: val, isBool, isCustom: !cat,
      placeholder: cat?.placeholder || 'Value...',
    })
    i++
  }
  return parsed
}

function serializeRows(): string {
  return rows.value
    .filter((r) => r.flag && r.flag !== '--')
    .map((r) => {
      if (r.isBool) return r.flag
      const v = (r.value || '').trim()
      return v ? (v.includes(' ') ? `${r.flag}="${v}"` : `${r.flag}=${v}`) : r.flag
    })
    .join(' ')
}

function sync() { emit('update:modelValue', serializeRows()) }

function isProcessGlobal(flagName: string): boolean {
  const item = RCLONE_FLAGS_CATALOG.find((c) => c.flag === flagName)
  return !!item?.isProcessGlobal
}

function getFlagLabel(flagName: string): string {
  const item = RCLONE_FLAGS_CATALOG.find((c) => c.flag === flagName)
  return item?.label || flagName
}

function getFlagDesc(flagName: string): string {
  const item = RCLONE_FLAGS_CATALOG.find((c) => c.flag === flagName)
  return item?.desc || 'Custom parameter'
}

const bisyncCount = computed(() => rows.value.filter((r) => getRowDomain(r) === 'bisync').length)
const rcloneCount = computed(() => rows.value.filter((r) => getRowDomain(r) === 'rclone').length)
const customCount = computed(() => rows.value.filter((r) => getRowDomain(r) === 'custom').length)

const filteredRows = computed(() => {
  if (activeFilter.value === 'all') return rows.value
  return rows.value.filter((r) => getRowDomain(r) === activeFilter.value)
})

function addBisyncFlag() {
  const target = bisyncCatalog.value[0]?.flag || '--track-renames'
  const cat = getFlagCatalogItem(target)
  rows.value.push({
    id: 'f_' + Math.random().toString(36).slice(2, 8),
    flag: target, value: '', isBool: !!cat?.isBool, isCustom: false,
    placeholder: cat?.placeholder || 'Value...',
  })
  activeFilter.value = 'all'
  sync()
}

function addRcloneFlag() {
  const target = rcloneCatalog.value[0]?.flag || '--checkers'
  const cat = getFlagCatalogItem(target)
  rows.value.push({
    id: 'f_' + Math.random().toString(36).slice(2, 8),
    flag: target, value: '', isBool: !!cat?.isBool, isCustom: false,
    placeholder: cat?.placeholder || 'Value...',
  })
  activeFilter.value = 'all'
  sync()
}

function addCustomFlag() {
  rows.value.push({
    id: 'f_' + Math.random().toString(36).slice(2, 8),
    flag: '--', value: '', isBool: false, isCustom: true,
    placeholder: 'Value...',
  })
  activeFilter.value = 'all'
  sync()
}

function removeRowById(id: string) {
  const idx = rows.value.findIndex((r) => r.id === id)
  if (idx !== -1) {
    rows.value.splice(idx, 1)
    sync()
  }
}

function onFlagSelect(rowId: string, val: string) {
  const r = rows.value.find((item) => item.id === rowId)
  if (!r) return
  if (val === 'custom') {
    r.isCustom = true; r.flag = '--'; r.isBool = false; r.value = ''; r.placeholder = 'Value...'
  } else {
    const cat = RCLONE_FLAGS_CATALOG.find((c) => c.flag === val)
    r.isCustom = false; r.flag = val; r.isBool = !!cat?.isBool; r.value = ''
    r.placeholder = cat?.placeholder || 'Value...'
  }
  sync()
}

onMounted(() => { rows.value = parseFlags(props.modelValue) })
watch(() => props.modelValue, (newVal) => {
  if (newVal !== serializeRows()) rows.value = parseFlags(newVal)
})
</script>


<template>
  <div class="space-y-3 p-3 bg-carbon-surface/80 border border-carbon-border rounded-xl">
    <!-- Header with Action Buttons -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
      <div class="flex items-center gap-2">
        <label class="text-xs font-semibold text-slate-200">
          {{ scopeFilter === 'engine' ? 'Daemon Engine & Concurrency Defaults' : (scopeFilter === 'all' ? 'Fleet Engine & Sync Flags' : 'Guided Flags Builder') }}
        </label>
        <span class="text-[10px] px-2 py-0.5 rounded font-mono bg-carbon-elevated text-slate-400 border border-carbon-border">
          {{ rows.length }} active
        </span>
      </div>

      <!-- Quick Add Buttons -->
      <div class="flex items-center gap-1.5 flex-wrap">
        <button
          type="button"
          @click="addBisyncFlag()"
          class="px-2 py-1 bg-sky-500/15 hover:bg-sky-500/25 border border-sky-500/30 text-sky-300 rounded-lg text-xs font-medium transition flex items-center gap-1 cursor-pointer"
          title="Add a standard Bisync synchronization behavior flag"
        >
          <i class="pi pi-plus text-[10px]"></i><span>Bisync Hub</span>
        </button>
        <button
          type="button"
          @click="addRcloneFlag()"
          class="px-2 py-1 bg-violet-500/15 hover:bg-violet-500/25 border border-violet-500/30 text-violet-300 rounded-lg text-xs font-medium transition flex items-center gap-1 cursor-pointer"
          title="Add a standard Rclone core concurrency, network, or rate limit flag"
        >
          <i class="pi pi-plus text-[10px]"></i><span>Rclone Core</span>
        </button>
        <button
          type="button"
          @click="addCustomFlag()"
          class="px-2 py-1 bg-carbon-surface hover:bg-carbon-elevated border border-carbon-border text-slate-300 hover:text-white rounded-lg text-xs font-medium transition flex items-center gap-1 cursor-pointer"
          title="Add an arbitrary user-defined custom flag"
        >
          <i class="pi pi-plus text-[10px]"></i><span>Custom</span>
        </button>
      </div>
    </div>

    <!-- Segmented Domain Filter Toolbar (shown when there are rows) -->
    <div v-if="rows.length > 0" class="flex items-center gap-1.5 p-1 bg-carbon-bg border border-carbon-border rounded-lg text-xs">
      <button
        type="button"
        @click="activeFilter = 'all'"
        :class="[
          'px-2 py-0.5 rounded text-[11px] font-medium transition cursor-pointer flex items-center gap-1.5',
          activeFilter === 'all' ? 'bg-carbon-elevated text-white shadow-xs' : 'text-slate-400 hover:text-slate-200'
        ]"
      >
        <span>All</span>
        <span class="text-[9px] px-1 rounded bg-carbon-surface border border-carbon-border font-mono">{{ rows.length }}</span>
      </button>

      <button
        type="button"
        @click="activeFilter = 'bisync'"
        :class="[
          'px-2 py-0.5 rounded text-[11px] font-medium transition cursor-pointer flex items-center gap-1.5',
          activeFilter === 'bisync' ? 'bg-sky-500/20 text-sky-200 border border-sky-500/30' : 'text-slate-400 hover:text-sky-300'
        ]"
      >
        <span class="w-1.5 h-1.5 rounded-full bg-sky-400"></span>
        <span>Bisync Hub</span>
        <span class="text-[9px] px-1 rounded bg-sky-500/10 font-mono text-sky-300">{{ bisyncCount }}</span>
      </button>

      <button
        type="button"
        @click="activeFilter = 'rclone'"
        :class="[
          'px-2 py-0.5 rounded text-[11px] font-medium transition cursor-pointer flex items-center gap-1.5',
          activeFilter === 'rclone' ? 'bg-violet-500/20 text-violet-200 border border-violet-500/30' : 'text-slate-400 hover:text-violet-300'
        ]"
      >
        <span class="w-1.5 h-1.5 rounded-full bg-violet-400"></span>
        <span>Rclone Core</span>
        <span class="text-[9px] px-1 rounded bg-violet-500/10 font-mono text-violet-300">{{ rcloneCount }}</span>
      </button>

      <button
        v-if="customCount > 0 || activeFilter === 'custom'"
        type="button"
        @click="activeFilter = 'custom'"
        :class="[
          'px-2 py-0.5 rounded text-[11px] font-medium transition cursor-pointer flex items-center gap-1.5',
          activeFilter === 'custom' ? 'bg-amber-500/20 text-amber-200 border border-amber-500/30' : 'text-slate-400 hover:text-amber-300'
        ]"
      >
        <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
        <span>Custom</span>
        <span class="text-[9px] px-1 rounded bg-amber-500/10 font-mono text-amber-300">{{ customCount }}</span>
      </button>
    </div>

    <!-- Empty States -->
    <div v-if="rows.length === 0" class="p-4 text-center rounded-lg border border-dashed border-carbon-border text-slate-500 text-xs">
      No synchronization or engine flags configured. Built-in defaults will be used.
    </div>
    <div v-else-if="filteredRows.length === 0" class="p-3 text-center rounded-lg border border-dashed border-carbon-border text-slate-500 text-xs">
      No {{ activeFilter === 'bisync' ? 'Bisync Hub' : (activeFilter === 'rclone' ? 'Rclone Core' : 'Custom') }} flags configured in active selection.
    </div>
    <div v-else class="space-y-2">
      <div
        v-for="row in filteredRows"
        :key="row.id"
        class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 p-2 bg-carbon-bg border border-carbon-border rounded-lg"
      >
        <!-- Domain Badge & Flag Select -->
        <div class="sm:w-1/2 min-w-[170px] flex items-center gap-1.5">
          <!-- Domain Tag -->
          <span
            v-if="getRowDomain(row) === 'bisync'"
            class="text-[9px] px-1.5 py-0.5 rounded bg-sky-500/15 border border-sky-500/30 text-sky-300 font-mono shrink-0"
            title="Bisync Hub operational synchronization flag"
          >
            Bisync
          </span>
          <span
            v-else-if="getRowDomain(row) === 'rclone'"
            class="text-[9px] px-1.5 py-0.5 rounded bg-violet-500/15 border border-violet-500/30 text-violet-300 font-mono shrink-0"
            title="Rclone Core Engine flag"
          >
            Rclone
          </span>
          <span
            v-else
            class="text-[9px] px-1.5 py-0.5 rounded bg-amber-500/15 border border-amber-500/30 text-amber-300 font-mono shrink-0"
            title="User custom flag"
          >
            Custom
          </span>

          <!-- Flag Selector Dropdown with Grouping -->
          <select
            :value="row.isCustom ? 'custom' : row.flag"
            @change="onFlagSelect(row.id, ($event.target as HTMLSelectElement).value)"
            class="w-full bg-carbon-surface border border-carbon-border rounded-md px-2 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
            :title="getFlagDesc(row.flag)"
          >
            <optgroup v-if="bisyncCatalog.length" label="Bisync Hub Flags (Sync & Conflicts)">
              <option v-for="c in bisyncCatalog" :key="c.flag" :value="c.flag">{{ c.label }}</option>
            </optgroup>
            <optgroup v-if="rcloneCatalog.length" label="Rclone Core Engine Flags (Concurrency & Net)">
              <option v-for="c in rcloneCatalog" :key="c.flag" :value="c.flag">{{ c.label }}</option>
            </optgroup>
            <optgroup label="User Defined">
              <option v-if="!row.isCustom && !availableCatalog.some(c => c.flag === row.flag)" :value="row.flag">{{ getFlagLabel(row.flag) }}</option>
              <option value="custom">Custom flag (--...)</option>
            </optgroup>
          </select>

          <!-- Process Global Badge -->
          <span
            v-if="isProcessGlobal(row.flag)"
            class="text-[9px] px-1.5 py-0.5 rounded bg-amber-500/15 border border-amber-500/30 text-amber-300 font-mono shrink-0 hidden sm:inline"
            title="Process-global in Rclone RC. Dynamically applied and queued cleanly."
          >
            Global
          </span>
        </div>

        <!-- Custom Flag Input -->
        <div v-if="row.isCustom" class="sm:w-1/4">
          <input
            v-model="row.flag"
            @input="sync()"
            type="text"
            placeholder="--custom-flag"
            class="w-full bg-carbon-surface border border-carbon-border rounded-md px-2 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <!-- Flag Value or Boolean Enabled Pill -->
        <div class="flex-1">
          <div
            v-if="row.isBool"
            class="flex items-center gap-1.5 px-2.5 py-1.5 bg-emerald-500/10 border border-emerald-500/20 rounded-md text-[11px] text-emerald-300 font-mono"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
            <span>Enabled</span>
          </div>
          <input
            v-else
            v-model="row.value"
            @input="sync()"
            type="text"
            :placeholder="row.placeholder || 'Value...'"
            class="w-full bg-carbon-surface border border-carbon-border rounded-md px-2 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <!-- Remove Button -->
        <button
          type="button"
          @click="removeRowById(row.id)"
          class="p-1.5 text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 rounded-md transition cursor-pointer self-end sm:self-center"
          title="Remove flag"
        >
          <i class="pi pi-trash text-xs"></i>
        </button>
      </div>
    </div>
    <div v-if="validationError" class="p-2.5 bg-rose-500/10 border border-rose-500/20 rounded-lg text-xs text-rose-300 flex items-center gap-2 font-mono">
      <i class="pi pi-exclamation-triangle text-rose-400"></i><span>{{ validationError }}</span>
    </div>
  </div>
</template>
