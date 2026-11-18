<template>
  <Dialog
    :visible="settingsStore.showSystemHealthModal"
    @update:visible="val => !val && settingsStore.closeSystemHealthModal()"
    modal
    header="System Health & Engine Center"
    :style="{ width: '92vw', maxWidth: '820px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border shadow-2xl rounded-2xl overflow-hidden' },
      header: { class: 'p-4 border-b border-carbon-border bg-carbon-surface text-white flex items-center justify-between' },
      content: { class: 'p-5 bg-carbon-canvas text-xs font-mono space-y-4 max-h-[75vh] overflow-y-auto' },
      footer: { class: 'p-3 border-t border-carbon-border bg-carbon-surface flex justify-between items-center' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div
          class="w-8 h-8 rounded-xl border flex items-center justify-center shrink-0"
          :class="isHealthy ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30' : 'bg-rose-500/10 text-rose-400 border-rose-500/30'"
        >
          <i class="pi pi-heart-fill text-base"></i>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-sm font-bold text-white tracking-wide">System Health & Engine Center</h3>
            <span
              class="text-[10px] uppercase font-mono px-2 py-0.5 rounded border"
              :class="isHealthy ? 'bg-emerald-950 text-emerald-300 border-emerald-800/60' : 'bg-rose-950 text-rose-300 border-rose-800/60'"
            >
              {{ isHealthy ? 'ONLINE' : 'DEGRADED' }}
            </span>
          </div>
          <p class="text-[11px] text-slate-400">
            Latency: <span class="font-bold text-white">{{ latencyText }}</span> | Mode: <span class="text-electric-blue font-bold">FSM v2</span>
          </p>
        </div>
      </div>
    </template>

    <!-- Tab Navigation -->
    <div class="flex items-center gap-1.5 border-b border-carbon-border pb-3">
      <button
        v-for="t in tabs"
        :key="t.id"
        type="button"
        @click="activeTab = t.id"
        :class="activeTab === t.id ? 'bg-electric-blue text-black font-bold shadow' : 'bg-carbon-surface text-slate-400 hover:text-white border border-carbon-border'"
        class="px-3 py-1.5 rounded-lg transition-colors cursor-pointer text-xs flex items-center gap-1.5"
      >
        <i :class="t.icon" class="text-xs"></i>
        <span>{{ t.label }}</span>
      </button>
    </div>

    <!-- TAB 1: DAEMON & DIAGNOSTICS -->
    <div v-if="activeTab === 'daemon'" class="space-y-4">
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Daemon</span>
          <span class="text-sm font-bold" :class="isHealthy ? 'text-emerald-400' : 'text-rose-400'">
            {{ isHealthy ? 'Healthy' : 'Degraded' }}
          </span>
        </div>

        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Latency</span>
          <span class="text-sm font-bold text-white">{{ latencyText }}</span>
        </div>

        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Engine State</span>
          <span class="text-sm font-bold" :class="telemetryStore.telemetry?.is_syncing ? 'text-amber-400' : 'text-slate-300'">
            {{ telemetryStore.telemetry?.is_syncing ? 'Syncing...' : 'Idle' }}
          </span>
        </div>

        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Safety Guard</span>
          <span class="text-sm font-bold text-emerald-400">Strict Jails</span>
        </div>
      </div>

      <!-- Health Degraded Diagnostic Card -->
      <div v-if="!isHealthy" class="p-4 bg-rose-500/10 border border-rose-500/30 rounded-xl space-y-2">
        <div class="flex items-center gap-2 text-rose-300 font-bold text-xs">
          <i class="pi pi-exclamation-triangle"></i>
          <span>Daemon Subsystem Degraded or Unreachable</span>
        </div>
        <p class="text-[11px] text-slate-300">
          The background daemon status is reported as <code class="text-rose-300 font-mono">{{ telemetryStore.telemetry?.daemon_health?.status || 'offline' }}</code>.
          Background polling and automatic scheduled syncs may be paused until Rclone RC connectivity is restored.
        </p>
        <div class="pt-1 flex items-center gap-2">
          <button
            type="button"
            @click="telemetryStore.fetchTelemetry"
            class="px-2.5 py-1 rounded bg-carbon-surface hover:bg-carbon-elevated border border-carbon-border text-[11px] text-white flex items-center gap-1.5 cursor-pointer"
          >
            <i class="pi pi-refresh text-xs"></i>
            <span>Retry Connection Probe</span>
          </button>
        </div>
      </div>

      <!-- Quick Operations Card -->
      <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h4 class="text-xs font-bold text-white">Live Daemon Logs & Run Correlator</h4>
          <p class="text-[11px] text-slate-400">Inspect interleaved stdout/stderr and Rclone RC JSON notices</p>
        </div>
        <button
          type="button"
          @click="openDaemonLogs"
          class="px-3.5 py-1.5 rounded-lg bg-electric-blue text-black font-bold text-xs cursor-pointer flex items-center gap-1.5 hover:bg-sky-400 transition-colors"
        >
          <i class="pi pi-terminal"></i>
          <span>View Correlated Logs</span>
        </button>
      </div>
    </div>

    <!-- TAB 2: MCP AGENT SERVER -->
    <div v-else-if="activeTab === 'mcp'" class="space-y-4">
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Agent Status</span>
          <span class="text-sm font-bold" :class="settingsStore.mcpStatus?.status === 'active' ? 'text-violet-400' : 'text-slate-400'">
            {{ settingsStore.mcpStatus?.status === 'active' ? 'Active (Ready)' : 'Offline' }}
          </span>
        </div>

        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Safety Mode</span>
          <span class="text-sm font-bold text-emerald-400">
            {{ settingsStore.mcpStatus?.read_only ? 'Read-Only Safe' : 'Read/Write Operations' }}
          </span>
        </div>

        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Available Tools</span>
          <span class="text-sm font-bold text-white">24 Standard Tools</span>
        </div>
      </div>

      <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h4 class="text-xs font-bold text-white">Model Context Protocol Inspector</h4>
          <p class="text-[11px] text-slate-400">Tool definitions, client configurations, and agent telemetry</p>
        </div>
        <button
          type="button"
          @click="settingsStore.closeSystemHealthModal(); settingsStore.openMcpModal()"
          class="px-3.5 py-1.5 rounded-lg bg-violet-600 text-white font-bold text-xs cursor-pointer flex items-center gap-1.5 hover:bg-violet-500 transition-colors"
        >
          <i class="pi pi-server"></i>
          <span>Open MCP Inspector</span>
        </button>
      </div>
    </div>

    <!-- TAB 3: REMOTE CONNECTIVITY -->
    <div v-else-if="activeTab === 'connectivity'" class="space-y-3">
      <div class="text-[11px] text-slate-400">Status and reachability of configured storage remotes.</div>

      <div v-if="remotesStore.remotes.length === 0" class="p-8 text-center text-slate-500">
        No remote backends configured.
      </div>

      <div
        v-for="remote in remotesStore.remotes"
        :key="remote.name"
        class="p-3 bg-carbon-surface border border-carbon-border rounded-xl flex items-center justify-between gap-3"
      >
        <div class="flex items-center gap-2.5">
          <div class="w-7 h-7 rounded-lg bg-carbon-canvas border border-carbon-border flex items-center justify-center text-slate-300">
            <i class="pi pi-cloud text-xs"></i>
          </div>
          <div>
            <div class="font-bold text-white text-xs font-mono">{{ remote.name }}</div>
            <div class="text-[10px] text-slate-500 uppercase">{{ remote.type }}</div>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <span
            class="text-[9px] uppercase font-mono px-2 py-0.5 rounded border"
            :class="remote.status === 'ok' ? 'bg-emerald-950 text-emerald-400 border-emerald-800' : 'bg-slate-800 text-slate-400 border-slate-700'"
          >
            {{ remote.status === 'ok' ? 'Connected' : (remote.status || 'Configured') }}
          </span>
          <button
            type="button"
            @click="remotesStore.checkAuth(remote.name)"
            class="px-2 py-1 rounded bg-carbon-canvas hover:bg-carbon-elevated border border-carbon-border text-[10px] text-slate-300 hover:text-white cursor-pointer"
          >
            Test
          </button>
        </div>
      </div>
    </div>

    <!-- TAB 4: SCHEDULER & POLLER -->
    <div v-else-if="activeTab === 'poller'" class="space-y-4">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-bold text-white">Live Telemetry Poller</span>
            <span
              class="text-[10px] font-mono px-2 py-0.5 rounded border"
              :class="appStore.isPollingPaused ? 'bg-amber-950/80 border-amber-600 text-amber-300' : 'bg-emerald-950/80 border-emerald-600 text-emerald-300'"
            >
              {{ appStore.isPollingPaused ? 'PAUSED' : 'ACTIVE' }}
            </span>
          </div>
          <p class="text-[11px] text-slate-400">
            Governs real-time status fetching across Profiles, FSM State, Transfers, and Queue.
          </p>
          <div class="pt-1">
            <button
              type="button"
              @click="appStore.togglePolling()"
              class="px-3 py-1.5 rounded-lg text-xs font-semibold cursor-pointer transition-colors flex items-center gap-2"
              :class="appStore.isPollingPaused ? 'bg-emerald-600 hover:bg-emerald-500 text-white' : 'bg-amber-600 hover:bg-amber-500 text-white'"
            >
              <i :class="appStore.isPollingPaused ? 'pi pi-play' : 'pi pi-pause'"></i>
              <span>{{ appStore.isPollingPaused ? 'Resume Poller (Alt+P)' : 'Pause Poller (Alt+P)' }}</span>
            </button>
          </div>
        </div>

        <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-2">
          <div class="text-xs font-bold text-white">Power & Resource Management</div>
          <p class="text-[11px] text-slate-400 leading-relaxed">
            Polling automatically suspends when the cockpit browser tab is hidden to conserve system battery and network bandwidth.
          </p>
          <div class="text-[11px] font-mono text-slate-400 pt-1">
            Standard Interval: <span class="text-white font-bold">3,000 ms</span>
          </div>
        </div>
      </div>

      <!-- Keyboard Shortcuts Reference Box -->
      <div class="p-3.5 bg-carbon-canvas border border-carbon-border rounded-xl space-y-2">
        <div class="text-xs font-bold text-slate-300">Global Cockpit Shortcuts</div>
        <div class="text-xs text-slate-400 font-mono grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1">
          <div><span class="text-indigo-400 font-bold">Alt + P</span> : Toggle Telemetry Polling</div>
          <div><span class="text-indigo-400 font-bold">C / Ctrl+K</span> : Global Create Menu</div>
          <div><span class="text-indigo-400 font-bold">N</span> : New Sync Profile</div>
          <div><span class="text-indigo-400 font-bold">R</span> : Connect Storage Remote</div>
          <div><span class="text-indigo-400 font-bold">L</span> : Toggle Log Dock</div>
          <div><span class="text-indigo-400 font-bold">? / F1</span> : Help & Diagnostics</div>
        </div>
      </div>
    </div>

    <template #footer>
      <button
        type="button"
        @click="openDiagnostics"
        class="px-3.5 py-1.5 rounded-xl bg-carbon-surface hover:bg-carbon-elevated border border-carbon-border text-electric-blue text-xs font-bold cursor-pointer flex items-center gap-1.5"
      >
        <i class="pi pi-shield"></i>
        <span>Run Full Diagnostic Audit</span>
      </button>

      <button
        type="button"
        @click="settingsStore.closeSystemHealthModal()"
        class="px-4 py-1.5 rounded-xl border border-carbon-border text-slate-300 hover:text-white cursor-pointer"
      >
        Close
      </button>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Dialog from 'primevue/dialog'
import { useSettingsStore } from '../stores/settings'
import { useTelemetryStore } from '../stores/telemetry'
import { useRemotesStore } from '../stores/remotes'
import { useLogsStore } from '../stores/logs'
import { useHelpStore } from '../stores/help'
import { useAppStore } from '../stores/app'

const settingsStore = useSettingsStore()
const telemetryStore = useTelemetryStore()
const remotesStore = useRemotesStore()
const logsStore = useLogsStore()
const helpStore = useHelpStore()
const appStore = useAppStore()

const activeTab = computed<'daemon' | 'mcp' | 'connectivity' | 'poller'>({
  get: () => settingsStore.systemHealthTab,
  set: (val) => { settingsStore.systemHealthTab = val },
})

const tabs = [
  { id: 'daemon' as const, label: 'Daemon & Diagnostics', icon: 'pi pi-server' },
  { id: 'mcp' as const, label: 'MCP Agent Server', icon: 'pi pi-bolt' },
  { id: 'connectivity' as const, label: 'Remote Connectivity', icon: 'pi pi-cloud' },
  { id: 'poller' as const, label: 'Scheduler & Poller', icon: 'pi pi-clock' },
]

const isHealthy = computed(() => telemetryStore.daemonHealthy)
const latencyText = computed(() => {
  const ms = telemetryStore.latencyMs
  return ms != null && ms >= 0 ? `${ms.toFixed(1)} ms` : '-- ms'
})

function openDiagnostics() {
  settingsStore.closeSystemHealthModal()
  helpStore.openDrawer('diagnostics', '', { id: 'view_telemetry', title: 'Engine & Telemetry' })
}

function openDaemonLogs() {
  settingsStore.closeSystemHealthModal()
  logsStore.openDaemonModal()
}
</script>
