<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { RCLONE_FLAGS_CATALOG, validateCustomFlagsString, type FlagCatalogItem } from '../data/flags-catalog'

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

const rows = ref<FlagRow[]>([])
const validationError = computed(() => validateCustomFlagsString(props.modelValue || ''))

const availableCatalog = computed<FlagCatalogItem[]>(() => {
  const t = props.remoteType?.toLowerCase() || 'all'
  return RCLONE_FLAGS_CATALOG.filter((c) => {
    const backendMatch = !c.backends || c.backends.includes('all') || c.backends.includes(t)
    const scopeMatch = props.scopeFilter === 'all' || c.scope === props.scopeFilter
    return backendMatch && scopeMatch
  })
})

const engineCatalog = computed(() => availableCatalog.value.filter((c) => c.scope === 'engine'))
const profileCatalog = computed(() => availableCatalog.value.filter((c) => c.scope === 'profile'))

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

function addRow() {
  const target = availableCatalog.value[0]?.flag || (props.scopeFilter === 'engine' ? '--checkers' : '--track-renames')
  const cat = RCLONE_FLAGS_CATALOG.find((c) => c.flag === target)
  rows.value.push({
    id: 'f_' + Math.random().toString(36).slice(2, 8),
    flag: target, value: '', isBool: !!cat?.isBool, isCustom: false,
    placeholder: cat?.placeholder || 'Value...',
  })
  sync()
}

function removeRow(idx: number) { rows.value.splice(idx, 1); sync() }

function onFlagSelect(idx: number, val: string) {
  const r = rows.value[idx]
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
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <label class="text-xs font-semibold text-slate-200">
          {{ scopeFilter === 'engine' ? 'Daemon Engine & Concurrency Defaults' : (scopeFilter === 'all' ? 'Global Fleet Flags & Concurrency' : 'Guided Bisync Flags Builder') }}
        </label>
        <span class="text-[10px] px-2 py-0.5 rounded font-mono bg-carbon-elevated text-slate-400 border border-carbon-border">
          {{ rows.length }} active
        </span>
      </div>
      <button
        type="button"
        @click="addRow()"
        class="px-2.5 py-1 bg-electric-blue/10 hover:bg-electric-blue/20 text-electric-blue rounded-lg text-xs font-medium border border-electric-blue/30 transition flex items-center gap-1 cursor-pointer"
      >
        <i class="pi pi-plus text-[10px]"></i><span>Add Flag</span>
      </button>
    </div>

    <div v-if="rows.length === 0" class="p-3 text-center rounded-lg border border-dashed border-carbon-border text-slate-500 text-xs">
      {{ scopeFilter === 'engine' ? 'No custom engine defaults configured. Rclone defaults will be used.' : 'No custom flags configured. Default bisync flags will be used.' }}
    </div>
    <div v-else class="space-y-2">
      <div
        v-for="(row, idx) in rows"
        :key="row.id"
        class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 p-2 bg-carbon-bg border border-carbon-border rounded-lg"
      >
        <div class="sm:w-1/3 min-w-[130px] flex items-center gap-1.5">
          <select
            :value="row.isCustom ? 'custom' : row.flag"
            @change="onFlagSelect(idx, ($event.target as HTMLSelectElement).value)"
            class="w-full bg-carbon-surface border border-carbon-border rounded-md px-2 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
            :title="getFlagDesc(row.flag)"
          >
            <template v-if="scopeFilter === 'all'">
              <optgroup v-if="profileCatalog.length" label="Profile Behaviors">
                <option v-for="c in profileCatalog" :key="c.flag" :value="c.flag">{{ c.label }}</option>
              </optgroup>
              <optgroup v-if="engineCatalog.length" label="Daemon & Engine Concurrency">
                <option v-for="c in engineCatalog" :key="c.flag" :value="c.flag">{{ c.label }}</option>
              </optgroup>
            </template>
            <template v-else>
              <option v-for="c in availableCatalog" :key="c.flag" :value="c.flag">{{ c.label }}</option>
            </template>
            <option v-if="!row.isCustom && !availableCatalog.some(c => c.flag === row.flag)" :value="row.flag">{{ getFlagLabel(row.flag) }}</option>
            <option value="custom">Custom flag (--...)</option>
          </select>
          <span
            v-if="isProcessGlobal(row.flag)"
            class="text-[9px] px-1.5 py-0.5 rounded bg-amber-500/15 border border-amber-500/30 text-amber-300 font-mono shrink-0 hidden sm:inline"
            title="Process-global in Rclone RC. Dynamically applied and queued cleanly."
          >
            Global
          </span>
          <span
            v-else-if="row.isCustom"
            class="text-[9px] px-1.5 py-0.5 rounded bg-slate-500/15 border border-slate-500/30 text-slate-300 font-mono shrink-0 hidden sm:inline"
            title="User-defined custom flag"
          >
            Custom
          </span>
        </div>

        <div v-if="row.isCustom" class="sm:w-1/4">
          <input
            v-model="row.flag"
            @input="sync()"
            type="text"
            placeholder="--custom-flag"
            class="w-full bg-carbon-surface border border-carbon-border rounded-md px-2 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <div class="flex-1">
          <div
            v-if="row.isBool"
            class="flex items-center gap-1.5 px-2.5 py-1.5 bg-emerald-500/10 border border-emerald-500/20 rounded-md text-[11px] text-emerald-300 font-mono"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span><span>Enabled</span>
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

        <button
          type="button"
          @click="removeRow(idx)"
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
