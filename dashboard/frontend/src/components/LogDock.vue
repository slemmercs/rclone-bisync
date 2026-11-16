<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted } from 'vue'
import { useLogsStore } from '../stores/logs'

interface ParsedLog {
  raw: string
  time: string
  level: string
  action?: string
  msg: string
}

const logsStore = useLogsStore()

type HeightPreset = 'collapsed' | 'docked' | 'expanded'
const heightPreset = ref<HeightPreset>('collapsed')
const activeFilter = ref<'all' | 'ERROR' | 'NOTICE' | 'INFO'>('all')
const autoScroll = ref<boolean>(true)
const logContainerRef = ref<HTMLDivElement | null>(null)

onMounted(() => {
  logsStore.fetchLogs()
})

function parseLog(raw: string): ParsedLog {
  let time = ''
  let level = 'INFO'
  let action: string | undefined
  let msg = raw

  // Check if JSON
  if (raw.startsWith('{') && raw.endsWith('}')) {
    try {
      const obj = JSON.parse(raw)
      return {
        raw,
        time: obj.time || obj.timestamp || '',
        level: (obj.level || 'INFO').toUpperCase(),
        action: obj.action,
        msg: obj.msg || obj.message || raw,
      }
    } catch {}
  }

  // Check standard rclone/bisync formats:
  // e.g. "2026/11/16 18:20:00 NOTICE: message"
  const match = raw.match(/^(\d{4}[/-]\d{2}[/-]\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+([A-Z]+):\s*(.*)$/)
  if (match) {
    time = match[1]
    level = match[2].toUpperCase()
    msg = match[3]
  } else if (raw.toUpperCase().includes('ERROR')) {
    level = 'ERROR'
  } else if (raw.toUpperCase().includes('NOTICE') || raw.toUpperCase().includes('WARN')) {
    level = 'NOTICE'
  }

  return { raw, time, level, action, msg }
}

const parsedLogs = computed<ParsedLog[]>(() => {
  return logsStore.logs.map(parseLog)
})

const filteredLogs = computed<ParsedLog[]>(() => {
  if (activeFilter.value === 'all') return parsedLogs.value
  return parsedLogs.value.filter((l) => l.level === activeFilter.value)
})

const lastLog = computed<ParsedLog | null>(() => {
  if (parsedLogs.value.length === 0) return null
  return parsedLogs.value[parsedLogs.value.length - 1]
})

function scrollToBottom() {
  if (!autoScroll.value || !logContainerRef.value) return
  nextTick(() => {
    if (logContainerRef.value) {
      logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
    }
  })
}

watch(
  () => logsStore.logs.length,
  () => {
    scrollToBottom()
  }
)

function togglePreset(preset: HeightPreset) {
  heightPreset.value = heightPreset.value === preset ? 'collapsed' : preset
  if (heightPreset.value !== 'collapsed') {
    scrollToBottom()
  }
}

function getLevelClass(level: string) {
  if (level === 'ERROR') return 'bg-red-950 text-red-400 border border-red-800'
  if (level === 'NOTICE' || level === 'WARN') return 'bg-amber-950 text-amber-400 border border-amber-800'
  if (level === 'DEBUG') return 'bg-carbon-800 text-carbon-500 border border-carbon-700'
  return 'bg-carbon-800 text-carbon-300 border border-carbon-700'
}
</script>

<template>
  <div
    class="fixed bottom-0 left-0 right-0 z-40 bg-carbon-900 border-t border-carbon-700 shadow-2xl transition-all duration-200 flex flex-col font-mono"
    :class="{
      'h-10': heightPreset === 'collapsed',
      'h-56': heightPreset === 'docked',
      'h-96': heightPreset === 'expanded',
    }"
  >
    <!-- Dock Header Toolbar -->
    <div class="px-4 py-2 bg-carbon-950 border-b border-carbon-800 flex items-center justify-between gap-3 text-xs select-none shrink-0">
      <div class="flex items-center gap-3">
        <button
          @click="togglePreset('docked')"
          class="flex items-center gap-2 font-bold text-carbon-200 hover:text-white transition-colors cursor-pointer"
        >
          <i class="pi pi-terminal text-electric-400"></i>
          <span>Execution Log Dock</span>
        </button>

        <span class="px-2 py-0.5 rounded text-[10px] bg-carbon-850 text-carbon-400 border border-carbon-700">
          {{ logsStore.logs.length }} records
        </span>

        <!-- Collapsed inline preview -->
        <span
          v-if="heightPreset === 'collapsed' && lastLog"
          class="hidden md:inline-block text-[11px] text-carbon-400 truncate max-w-xl font-mono cursor-pointer"
          @click="togglePreset('docked')"
        >
          <span class="text-carbon-500">{{ lastLog.time || '' }}</span>
          <span class="mx-1.5 text-carbon-600">|</span>
          <span :class="lastLog.level === 'ERROR' ? 'text-red-400' : 'text-carbon-300'">{{ lastLog.msg }}</span>
        </span>
      </div>

      <!-- Controls & Filter Pills -->
      <div class="flex items-center gap-2">
        <div v-if="heightPreset !== 'collapsed'" class="flex items-center bg-carbon-900 p-0.5 rounded border border-carbon-700 text-[10px]">
          <button
            @click="activeFilter = 'all'"
            class="px-2 py-0.5 rounded transition-all"
            :class="activeFilter === 'all' ? 'bg-electric-500 text-white font-bold' : 'text-carbon-400 hover:text-white'"
          >
            All
          </button>
          <button
            @click="activeFilter = 'ERROR'"
            class="px-2 py-0.5 rounded transition-all"
            :class="activeFilter === 'ERROR' ? 'bg-red-600 text-white font-bold' : 'text-carbon-400 hover:text-white'"
          >
            Errors
          </button>
          <button
            @click="activeFilter = 'NOTICE'"
            class="px-2 py-0.5 rounded transition-all"
            :class="activeFilter === 'NOTICE' ? 'bg-amber-600 text-white font-bold' : 'text-carbon-400 hover:text-white'"
          >
            Notice
          </button>
        </div>

        <button
          v-if="heightPreset !== 'collapsed'"
          @click="autoScroll = !autoScroll"
          class="px-2 py-0.5 rounded text-[10px] border transition-colors"
          :class="autoScroll ? 'bg-electric-950 text-electric-300 border-electric-600' : 'bg-carbon-850 text-carbon-500 border-carbon-700'"
        >
          Auto-scroll
        </button>

        <button
          @click="logsStore.fetchLogs()"
          class="p-1 rounded bg-carbon-850 hover:bg-carbon-800 text-carbon-400 hover:text-white"
          title="Refresh logs"
        >
          <i :class="logsStore.loading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
        </button>

        <button
          @click="togglePreset('docked')"
          class="p-1 rounded bg-carbon-850 hover:bg-carbon-800 text-carbon-400 hover:text-white"
          :title="heightPreset === 'docked' ? 'Collapse' : 'Docked (220px)'"
        >
          <i class="pi pi-window-minimize text-[10px]"></i>
        </button>
        <button
          @click="togglePreset('expanded')"
          class="p-1 rounded bg-carbon-850 hover:bg-carbon-800 text-carbon-400 hover:text-white"
          :title="heightPreset === 'expanded' ? 'Collapse' : 'Expanded (380px)'"
        >
          <i class="pi pi-window-maximize text-[10px]"></i>
        </button>
      </div>
    </div>

    <!-- Dock Virtual Stream Body -->
    <div
      v-if="heightPreset !== 'collapsed'"
      ref="logContainerRef"
      class="flex-1 overflow-y-auto p-3 divide-y divide-carbon-800/40 text-[11px] bg-carbon-950"
    >
      <div
        v-for="(log, idx) in filteredLogs"
        :key="idx"
        class="py-1 flex items-start gap-3 hover:bg-carbon-900/60 px-2 rounded font-mono"
      >
        <span class="text-carbon-500 shrink-0 text-[10px]">{{ log.time || '--:--:--' }}</span>
        <span class="px-1.5 py-0.2 rounded text-[9px] font-bold uppercase shrink-0" :class="getLevelClass(log.level)">
          {{ (log.level || 'INFO').trim() }}
        </span>
        <span v-if="log.action" class="px-1.5 py-0.2 rounded text-[9px] font-bold uppercase shrink-0 bg-[#1E222B] border border-[#333947] text-[#CBD5E1]">
          {{ log.action }}
        </span>
        <span class="text-carbon-300 break-all flex-1">{{ log.msg }}</span>
      </div>
      <div v-if="filteredLogs.length === 0" class="p-6 text-center text-carbon-500 text-xs">
        No log entries matching filter.
      </div>
    </div>
  </div>
</template>
