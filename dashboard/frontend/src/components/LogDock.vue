<template>
  <div
    class="fixed bottom-0 left-0 right-0 z-40 bg-carbon-900 border-t border-carbon-700 shadow-2xl transition-all duration-200 flex flex-col font-mono"
    :class="{
      'h-10': logsStore.dockHeightPreset === 'collapsed',
      'h-60': logsStore.dockHeightPreset === 'docked',
      'h-96': logsStore.dockHeightPreset === 'expanded',
    }"
  >
    <!-- Dock Header Toolbar -->
    <div class="px-4 py-2 bg-carbon-950 border-b border-carbon-800 flex items-center justify-between gap-3 text-xs select-none shrink-0">
      <div class="flex items-center gap-3">
        <button
          @click="logsStore.toggleDock('docked')"
          class="flex items-center gap-2 font-bold text-carbon-200 hover:text-white transition-colors cursor-pointer"
          title="Toggle Dock"
        >
          <i class="pi pi-terminal text-electric-400"></i>
          <span>Bisync Log Dock</span>
          <span class="px-1.5 py-0.2 rounded bg-carbon-800 border border-carbon-700 text-carbon-400 text-[10px]">
            {{ logsStore.logs.length }}
          </span>
        </button>

        <!-- Level Filter Pills -->
        <div v-if="logsStore.isDockOpen" class="flex items-center gap-1 bg-carbon-900 p-0.5 rounded border border-carbon-800 text-[10px]">
          <button
            v-for="lvl in ['ALL', 'ERROR', 'NOTICE', 'INFO']"
            :key="lvl"
            @click="logsStore.filterLevel = lvl"
            class="px-2 py-0.5 rounded transition-colors"
            :class="logsStore.filterLevel === lvl ? 'bg-electric-600 text-white font-semibold' : 'text-carbon-400 hover:text-carbon-200'"
          >
            {{ lvl }}
          </button>
        </div>

        <!-- Daemon Stream Trigger -->
        <button
          v-if="logsStore.isDockOpen"
          @click="logsStore.openDaemonModal(null)"
          class="px-2 py-0.5 rounded bg-electric-500/10 hover:bg-electric-500/20 text-electric-400 border border-electric-500/20 text-[10px] transition-colors flex items-center gap-1 cursor-pointer"
          title="Open Correlated Daemon Log Modal"
        >
          <i class="pi pi-external-link text-[9px]"></i>
          <span>Daemon Stream</span>
        </button>
      </div>


      <div class="flex items-center gap-2 text-carbon-400 text-[11px]">
        <!-- Auto-scroll Toggle -->
        <label v-if="logsStore.isDockOpen" class="flex items-center gap-1.5 cursor-pointer text-carbon-400 hover:text-carbon-200">
          <input type="checkbox" v-model="autoScroll" class="rounded bg-carbon-800 border-carbon-700 text-electric-500 text-xs" />
          <span>Auto-scroll</span>
        </label>

        <!-- Height Preset Buttons -->
        <button
          v-if="logsStore.dockHeightPreset !== 'expanded'"
          @click="logsStore.dockHeightPreset = 'expanded'"
          class="p-1 hover:text-white cursor-pointer"
          title="Expand"
        >
          <i class="pi pi-window-maximize text-xs"></i>
        </button>
        <button
          v-if="logsStore.dockHeightPreset !== 'docked'"
          @click="logsStore.dockHeightPreset = 'docked'"
          class="p-1 hover:text-white cursor-pointer"
          title="Dock"
        >
          <i class="pi pi-window-minimize text-xs"></i>
        </button>
        <button
          @click="logsStore.toggleDock('collapsed')"
          class="p-1 hover:text-white cursor-pointer"
          :title="logsStore.isDockOpen ? 'Collapse' : 'Open'"
        >
          <i :class="logsStore.isDockOpen ? 'pi pi-chevron-down' : 'pi pi-chevron-up'" class="text-xs"></i>
        </button>
      </div>
    </div>

    <!-- Pinned Log Stream Content -->
    <div
      v-if="logsStore.isDockOpen"
      ref="logContainerRef"
      class="flex-1 overflow-y-auto p-3 space-y-1 text-xs select-text leading-relaxed bg-carbon-950 font-mono"
    >
      <div v-if="logsStore.filteredLogs.length === 0" class="py-6 text-center text-carbon-500">
        No log entries matching filter criteria.
      </div>
      <div
        v-for="(log, idx) in logsStore.filteredLogs"
        :key="idx"
        class="flex items-start gap-2 hover:bg-carbon-900/60 px-1.5 py-0.5 rounded transition-colors"
      >
        <span class="text-carbon-500 text-[10px] select-none shrink-0">{{ idx + 1 }}</span>
        <button
          v-if="extractRunId(log)"
          type="button"
          @click.stop="logsStore.openDaemonModal({ run_id: extractRunId(log)! })"
          class="text-[9px] px-1.5 py-0.2 rounded bg-carbon-800 hover:bg-indigo-950 text-electric-400 hover:text-white border border-carbon-700 hover:border-indigo-600 shrink-0 transition-colors cursor-pointer"
          :title="'Open correlated daemon logs for run ' + extractRunId(log)"
        >
          run:{{ extractRunId(log) }}
        </button>
        <span class="text-carbon-200 break-all select-text">{{ log }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue'
import { useLogsStore } from '../stores/logs'

const logsStore = useLogsStore()
const autoScroll = ref<boolean>(true)
const logContainerRef = ref<HTMLDivElement | null>(null)

function extractRunId(log: string): string | null {
  const m = log.match(/\b(?:run[:=]|run_id[:=]|job[:=])([a-zA-Z0-9_-]+)\b/i)
  return m ? m[1] : null
}


onMounted(() => {
  logsStore.fetchLogs()
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
</script>
