<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '../../stores/app'
import { useSettingsStore } from '../../stores/settings'
import { useTelemetryStore } from '../../stores/telemetry'
import { useToastNotification } from '../../composables/useToastNotification'

const appStore = useAppStore()
const settingsStore = useSettingsStore()
const telemetryStore = useTelemetryStore()
const toast = useToastNotification()

const daemonStatus = computed(() => {
  return telemetryStore.daemonHealthy ? 'healthy' : 'degraded'
})

const isMcpActive = computed(() => {
  return settingsStore.mcpStatus?.status === 'active' || telemetryStore.telemetry?.mcp_status?.status === 'active'
})

const mcpStatusText = computed(() => {
  return isMcpActive.value ? 'MCP Online' : 'MCP Standby'
})

function handleTogglePolling() {
  const paused = appStore.togglePolling()
  if (paused) {
    toast.info('Telemetry Polling Paused', 'Press Alt+P or click indicator to resume live polling.')
  } else {
    toast.success('Telemetry Polling Resumed', 'Live telemetry and scheduler active.')
  }
}
</script>

<template>
  <div class="flex items-center gap-3">
    <div class="h-9 w-9 rounded-lg bg-electric-950 border border-electric-600 flex items-center justify-center text-electric-400 font-bold shadow-inner">
      <i class="pi pi-sync text-base"></i>
    </div>
    <div>
      <div class="flex items-center gap-2">
        <span class="font-bold text-carbon-50 text-base tracking-tight">Bisync Hub</span>
        <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-carbon-800 text-carbon-400 border border-carbon-700">FSM v2</span>
      </div>
      <div class="flex items-center gap-2 text-xs text-carbon-400">
        <span>Storage Control Plane</span>
        <span class="inline-block w-1 h-1 rounded-full bg-carbon-600"></span>
        <button
          type="button"
          @click="settingsStore.openSystemHealthModal()"
          class="flex items-center gap-1 font-mono text-[11px] cursor-pointer hover:underline"
          :class="daemonStatus === 'healthy' ? 'text-emerald-400' : 'text-amber-400'"
          title="Open System Health & Engine Center"
        >
          <span class="w-1.5 h-1.5 rounded-full" :class="daemonStatus === 'healthy' ? 'bg-emerald-400' : 'bg-amber-400 animate-pulse'"></span>
          {{ daemonStatus === 'healthy' ? 'RC Daemon Online' : 'Daemon Standby' }}
        </button>
        <span class="inline-block w-1 h-1 rounded-full bg-carbon-600"></span>
        <button
          type="button"
          @click="settingsStore.openMcpModal()"
          class="flex items-center gap-1 font-mono text-[11px] cursor-pointer hover:underline"
          :class="isMcpActive ? 'text-violet-400' : 'text-slate-500'"
          title="Open MCP Inspector"
        >
          <span class="w-1.5 h-1.5 rounded-full" :class="isMcpActive ? 'bg-violet-400' : 'bg-slate-600'"></span>
          <span>{{ mcpStatusText }}</span>
        </button>
        <span class="inline-block w-1 h-1 rounded-full bg-carbon-600"></span>
        <button
          type="button"
          @click="handleTogglePolling"
          class="flex items-center gap-1 font-mono text-[11px] cursor-pointer hover:underline"
          :class="appStore.isPollingPaused ? 'text-amber-400' : 'text-emerald-400'"
          title="Toggle Live Telemetry Poller (Alt+P)"
        >
          <span class="w-1.5 h-1.5 rounded-full" :class="appStore.isPollingPaused ? 'bg-amber-400' : 'bg-emerald-400 animate-pulse'"></span>
          <span>{{ appStore.isPollingPaused ? 'Polling Paused' : 'Live' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
