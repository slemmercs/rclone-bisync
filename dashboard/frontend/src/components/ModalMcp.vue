<template>
  <Dialog
    :visible="settingsStore.showMcpModal"
    @update:visible="val => !val && settingsStore.closeMcpModal()"
    modal
    header="Model Context Protocol (MCP) Inspector"
    :style="{ width: '92vw', maxWidth: '800px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border shadow-2xl rounded-2xl overflow-hidden' },
      header: { class: 'p-4 border-b border-carbon-border bg-carbon-surface text-white flex items-center justify-between' },
      content: { class: 'p-5 bg-carbon-canvas text-xs font-mono space-y-4 max-h-[75vh] overflow-y-auto' },
      footer: { class: 'p-3 border-t border-carbon-border bg-carbon-surface flex justify-end' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-xl bg-violet-500/10 border border-violet-500/30 text-violet-400 flex items-center justify-center shrink-0">
          <i class="pi pi-server text-base"></i>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-sm font-bold text-white tracking-wide">Model Context Protocol (MCP)</h3>
            <span
              class="text-[10px] uppercase font-mono px-2 py-0.5 rounded border"
              :class="isMcpActive ? 'bg-violet-950 text-violet-300 border-violet-800/60' : 'bg-slate-800 text-slate-400 border-slate-700/60'"
            >
              {{ isMcpActive ? 'ACTIVE' : 'OFFLINE' }}
            </span>
            <span
              class="text-[10px] uppercase font-mono px-2 py-0.5 rounded border"
              :class="isReadOnly ? 'bg-amber-950 text-amber-300 border-amber-800/60' : 'bg-emerald-950 text-emerald-300 border-emerald-800/60'"
            >
              {{ isReadOnly ? 'READ-ONLY' : 'READ/WRITE' }}
            </span>
          </div>
          <p class="text-[11px] text-slate-400">Agentic synchronization, conflict review inspection, and diagnostics</p>
        </div>
      </div>
    </template>

    <!-- Tab Pills -->
    <div class="flex items-center gap-1.5 border-b border-carbon-border pb-3">
      <button
        v-for="t in tabs"
        :key="t.id"
        type="button"
        @click="activeTab = t.id"
        :class="activeTab === t.id ? 'bg-violet-600 text-white font-bold shadow' : 'bg-carbon-surface text-slate-400 hover:text-white border border-carbon-border'"
        class="px-3 py-1.5 rounded-lg transition-colors cursor-pointer text-xs flex items-center gap-1.5"
      >
        <i :class="t.icon" class="text-xs"></i>
        <span>{{ t.label }}</span>
      </button>
    </div>

    <!-- TAB 1: SERVER STATUS & OVERVIEW -->
    <div v-if="activeTab === 'overview'" class="space-y-4">
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Agent Status</span>
          <span class="text-sm font-bold" :class="isMcpActive ? 'text-violet-400' : 'text-slate-400'">
            {{ isMcpActive ? 'Active (Ready)' : 'Offline' }}
          </span>
        </div>

        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Safety Mode</span>
          <span class="text-sm font-bold" :class="isReadOnly ? 'text-amber-400' : 'text-emerald-400'">
            {{ isReadOnly ? 'Read-Only Safe' : 'Read/Write Operations' }}
          </span>
        </div>

        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl">
          <span class="text-[10px] uppercase text-slate-500 block">Transport</span>
          <span class="text-sm font-bold text-white uppercase">
            {{ settingsStore.mcpStatus?.transport || 'stdio' }}
          </span>
        </div>
      </div>

      <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-2">
        <h4 class="text-xs font-bold text-white uppercase tracking-wider">Protocol Capabilities</h4>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center text-[11px] pt-1">
          <div class="p-2 rounded bg-carbon-canvas border border-carbon-border">
            <div class="text-white font-bold">24 Tools</div>
            <div class="text-[9px] text-slate-500 uppercase">Registered</div>
          </div>
          <div class="p-2 rounded bg-carbon-canvas border border-carbon-border">
            <div class="text-emerald-400 font-bold">Jailed</div>
            <div class="text-[9px] text-slate-500 uppercase">Containment</div>
          </div>
          <div class="p-2 rounded bg-carbon-canvas border border-carbon-border">
            <div class="text-electric-blue font-bold">Canary</div>
            <div class="text-[9px] text-slate-500 uppercase">Safeguards</div>
          </div>
          <div class="p-2 rounded bg-carbon-canvas border border-carbon-border">
            <div class="text-violet-400 font-bold">SHA-256</div>
            <div class="text-[9px] text-slate-500 uppercase">Audit Trail</div>
          </div>
        </div>
      </div>
    </div>

    <!-- TAB 2: CLIENT INTEGRATION -->
    <div v-else-if="activeTab === 'config'" class="space-y-4">
      <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-xs font-bold text-white uppercase tracking-wider">Client Configuration</span>
          <button
            type="button"
            @click="copyConfig"
            class="text-xs text-slate-300 hover:text-white flex items-center gap-1.5 px-2.5 py-1 bg-carbon-canvas rounded-lg border border-carbon-border cursor-pointer"
          >
            <i class="pi" :class="copiedConfig ? 'pi-check text-emerald-400' : 'pi-copy'"></i>
            <span>{{ copiedConfig ? 'Copied JSON!' : 'Copy Config' }}</span>
          </button>
        </div>
        <p class="text-[11px] text-slate-400">Add the snippet below to your MCP client config (e.g., Claude Desktop, Cursor, Cline):</p>
        <pre class="p-3 bg-carbon-canvas border border-carbon-border rounded-xl text-emerald-400 select-all overflow-x-auto leading-relaxed">{{ mcpClientSnippet }}</pre>
      </div>

      <div class="p-3.5 bg-violet-950/20 border border-violet-800/40 rounded-xl text-slate-300 space-y-1">
        <div class="font-bold text-violet-300">Confirmation Ticket Guard</div>
        <p class="text-[11px] text-slate-400 leading-relaxed">
          High blast-radius operations (--resync, --force, mass deletes, profile rollbacks) require a cryptographic confirmation token returned by the server with a 5-minute time-to-live before execution is permitted.
        </p>
      </div>
    </div>

    <!-- TAB 3: TOOL CATALOG -->
    <div v-else-if="activeTab === 'tools'" class="space-y-4 pr-1">
      <!-- Search filter -->
      <div class="relative">
        <input
          type="text"
          v-model="toolSearch"
          placeholder="Filter tools by name, description, or mode..."
          class="w-full bg-carbon-surface border border-carbon-border rounded-xl pl-9 pr-3 py-2 text-xs text-white placeholder-slate-500 font-mono focus:outline-none focus:border-violet-500"
        />
        <i class="pi pi-search text-slate-500 absolute left-3 top-3 text-xs pointer-events-none"></i>
      </div>

      <!-- Inspection Tools -->
      <div class="space-y-2">
        <div class="flex items-center gap-2">
          <span class="text-xs font-bold text-slate-300 uppercase tracking-wider">Inspection Tools</span>
          <span class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-emerald-950 text-emerald-400 border border-emerald-800">
            {{ inspectionTools.length }} Read-Only
          </span>
        </div>
        <div class="space-y-1.5">
          <div
            v-for="tool in inspectionTools"
            :key="tool.name"
            class="p-2.5 bg-carbon-surface border border-carbon-border rounded-xl flex items-start justify-between gap-3"
          >
            <div class="min-w-0">
              <span class="font-mono text-xs font-bold text-violet-300">{{ tool.name }}</span>
              <p class="text-[11px] text-slate-400 mt-0.5">{{ tool.desc }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Mutation Tools -->
      <div class="space-y-2 pt-2">
        <div class="flex items-center gap-2">
          <span class="text-xs font-bold text-slate-300 uppercase tracking-wider">Mutation & Execution Tools</span>
          <span class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-rose-950 text-rose-400 border border-rose-800">
            {{ mutationTools.length }} Guarded
          </span>
        </div>
        <div class="space-y-1.5">
          <div
            v-for="tool in mutationTools"
            :key="tool.name"
            class="p-2.5 bg-carbon-surface border border-carbon-border rounded-xl flex items-start justify-between gap-3"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-mono text-xs font-bold text-rose-300">{{ tool.name }}</span>
                <span v-if="tool.confirmed" class="text-[9px] uppercase font-mono px-1.5 py-0.2 rounded bg-amber-950 text-amber-400 border border-amber-800">
                  Confirmation Ticket
                </span>
              </div>
              <p class="text-[11px] text-slate-400 mt-0.5">{{ tool.desc }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <button
        type="button"
        @click="settingsStore.closeMcpModal()"
        class="px-4 py-1.5 rounded-xl border border-carbon-border text-slate-300 hover:text-white cursor-pointer"
      >
        Close
      </button>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import { useSettingsStore } from '../stores/settings'
import { useTelemetryStore } from '../stores/telemetry'

const settingsStore = useSettingsStore()
const telemetryStore = useTelemetryStore()
const activeTab = ref<'overview' | 'config' | 'tools'>('overview')
const copiedConfig = ref<boolean>(false)

const tabs = [
  { id: 'overview' as const, label: 'Server Status', icon: 'pi pi-info-circle' },
  { id: 'config' as const, label: 'Client Integration', icon: 'pi pi-code' },
  { id: 'tools' as const, label: 'Tool Catalog (24)', icon: 'pi pi-th-large' },
]

const isMcpActive = computed(() => {
  return settingsStore.mcpStatus?.status === 'active' || telemetryStore.telemetry?.mcp_status?.status === 'active'
})

const isReadOnly = computed(() => {
  return Boolean(settingsStore.mcpStatus?.read_only ?? telemetryStore.telemetry?.mcp_status?.read_only)
})

const mcpClientSnippet = computed(() => {
  return JSON.stringify({
    mcpServers: {
      'rclone-bisync': {
        command: 'rclone-bisync',
        args: ['mcp'],
        env: {}
      }
    }
  }, null, 2)
})

async function copyConfig() {
  try {
    await navigator.clipboard.writeText(mcpClientSnippet.value)
    copiedConfig.value = true
    setTimeout(() => { copiedConfig.value = false }, 2000)
  } catch (e) {
    console.error('Failed to copy MCP config:', e)
  }
}

const toolSearch = ref('')

const toolsList = [
  // Inspection Tools (10)
  { name: 'bisync_list_providers', desc: 'List supported cloud storage providers with categorized catalog and OAuth capabilities.', category: 'inspection' },
  { name: 'bisync_list_profiles', desc: 'List configured bisync profiles with current status, schedules, and remote path mappings.', category: 'inspection' },
  { name: 'bisync_list_remotes', desc: 'List configured Rclone remotes, storage types, and connectivity status.', category: 'inspection' },
  { name: 'bisync_explore_path', desc: 'Safely explore files and directories under local or remote path with jail containment.', category: 'inspection' },
  { name: 'bisync_preview_file', desc: 'Inspect safe text or image preview metadata for a file under remote or local path.', category: 'inspection' },
  { name: 'bisync_get_sync_history', desc: 'Query historical sync execution records and audit ledger.', category: 'inspection' },
  { name: 'bisync_get_status', desc: 'Get compact system health summary (status, active syncs, daemon health, canary status).', category: 'inspection' },
  { name: 'bisync_get_logs', desc: 'Retrieve recent bisync execution logs with tail limit, severity, and run ID filtering.', category: 'inspection' },
  { name: 'bisync_inspect_conflict_session', desc: 'Retrieve pending conflict review session details, diffs, and candidate resolutions.', category: 'inspection' },
  { name: 'bisync_validate_flags', desc: 'Validate CLI flags and preview compiled Rclone bisync parameters.', category: 'inspection' },

  // Mutation Tools (14)
  { name: 'bisync_create_profile', desc: 'Create a new bidirectional synchronization profile with path validation.', category: 'mutation' },
  { name: 'bisync_update_profile', desc: 'Update configuration settings on an existing sync profile.', category: 'mutation' },
  { name: 'bisync_delete_profile', desc: 'Permanently delete a sync profile with confirmation token protection.', category: 'mutation', confirmed: true },
  { name: 'bisync_rollback_profile', desc: 'Roll back a sync profile to a previous version snapshot from configuration history.', category: 'mutation', confirmed: true },
  { name: 'bisync_create_remote', desc: 'Provision a new Rclone remote target with optional profile provisioning bridge.', category: 'mutation' },
  { name: 'bisync_delete_remote', desc: 'Delete an existing Rclone remote with blast-radius protection and link safeguards.', category: 'mutation', confirmed: true },
  { name: 'bisync_trigger_sync', desc: 'Trigger synchronization on a profile with blast-radius protection and dry-run safety.', category: 'mutation', confirmed: true },
  { name: 'bisync_dedupe_by_hash', desc: 'Execute Rclone deduplication on a remote or profile path.', category: 'mutation', confirmed: true },
  { name: 'bisync_mkdir', desc: 'Create a directory under local or remote workspace path.', category: 'mutation' },
  { name: 'bisync_copy_item', desc: 'Copy a file or directory between remote and local storage targets.', category: 'mutation' },
  { name: 'bisync_move_item', desc: 'Move or rename a file or directory between remote and local storage targets.', category: 'mutation' },
  { name: 'bisync_delete_item', desc: 'Safely delete a file or directory with canary protection and confirmation safeguards.', category: 'mutation', confirmed: true },
  { name: 'bisync_purge_dir', desc: 'Recursively delete a directory and all contents with high blast-radius safeguards.', category: 'mutation', confirmed: true },
  { name: 'bisync_resolve_review', desc: 'Stage or apply conflict resolution decisions on a review session.', category: 'mutation', confirmed: true },
]

const filteredTools = computed(() => {
  const q = toolSearch.value.toLowerCase().trim()
  if (!q) return toolsList
  return toolsList.filter(t => t.name.toLowerCase().includes(q) || t.desc.toLowerCase().includes(q))
})

const inspectionTools = computed(() => filteredTools.value.filter(t => t.category === 'inspection'))
const mutationTools = computed(() => filteredTools.value.filter(t => t.category === 'mutation'))

</script>
