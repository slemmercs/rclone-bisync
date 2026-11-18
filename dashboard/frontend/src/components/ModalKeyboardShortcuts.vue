<script setup lang="ts">
import Dialog from 'primevue/dialog'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()

interface ShortcutItem {
  key: string
  label: string
  description: string
}

interface ShortcutGroup {
  name: string
  shortcuts: ShortcutItem[]
}

const groups: ShortcutGroup[] = [
  {
    name: 'Operator & Polling Controls',
    shortcuts: [
      { key: 'Alt + P', label: 'Toggle Polling', description: 'Pause or resume background telemetry and scheduled checks' },
      { key: 'Alt + R', label: 'Conflict Review', description: 'Open the sync conflict and circuit breaker review modal' },
      { key: 'C or Ctrl+K', label: 'Create Menu', description: 'Open the quick action create menu dropdown' },
      { key: 'Esc', label: 'Dismiss Overlays', description: 'Close any active modal dialog, drawer, or quick menu' },
    ],
  },
  {
    name: 'Quick Provisioning',
    shortcuts: [
      { key: 'N', label: 'New Profile', description: 'Open the bidirectional sync profile creator dialog' },
      { key: 'R', label: 'Connect Remote', description: 'Launch the storage remote provisioning wizard' },
      { key: 'L', label: 'Toggle Logs', description: 'Expand or collapse the live pinned log stream dock' },
    ],
  },
  {
    name: 'Workspace Navigation',
    shortcuts: [
      { key: '1', label: 'Fleet Overview', description: 'Switch active view to Sync Profiles Fleet' },
      { key: '2', label: 'Telemetry & Engine', description: 'Switch active view to Engine Telemetry' },
      { key: '3', label: 'Storage Remotes', description: 'Switch active view to Storage Remotes' },
      { key: '4', label: 'File Explorer', description: 'Switch active view to Dual-Pane Explorer' },
      { key: '5', label: 'Audit Trail', description: 'Switch active view to Ledger Audit Trail' },
      { key: '? or F1', label: 'Help & Reference', description: 'Open the interactive documentation and diagnostics drawer' },
    ],
  },
]
</script>

<template>
  <Dialog
    :visible="appStore.shortcutsModalOpen"
    @update:visible="(val) => (!val ? appStore.closeShortcutsModal() : null)"
    modal
    header="Cockpit Keyboard Shortcuts"
    :style="{ width: '560px', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-900 border border-carbon-700 rounded-2xl shadow-2xl text-carbon-50 overflow-hidden' },
      header: { class: 'bg-carbon-900 px-6 py-4 border-b border-carbon-800 text-carbon-50' },
      content: { class: 'p-6 bg-carbon-900 space-y-5 text-sm max-h-[75vh] overflow-y-auto' },
    }"
  >
    <div class="space-y-6">
      <div v-for="group in groups" :key="group.name" class="space-y-2.5">
        <h4 class="text-xs uppercase font-mono font-bold tracking-wider text-carbon-400">
          {{ group.name }}
        </h4>
        <div class="space-y-2 bg-carbon-950/70 p-3 rounded-xl border border-carbon-800">
          <div
            v-for="s in group.shortcuts"
            :key="s.key"
            class="flex items-center justify-between gap-3 text-xs"
          >
            <span class="text-carbon-300 font-medium">{{ s.label }}</span>
            <div class="flex items-center gap-2">
              <span class="text-[11px] text-carbon-500 hidden sm:inline">{{ s.description }}</span>
              <kbd class="px-2 py-0.5 rounded bg-carbon-850 text-electric-300 font-mono text-[11px] border border-carbon-700 shadow-sm shrink-0">
                {{ s.key }}
              </kbd>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="p-3 border-t border-carbon-800 bg-carbon-950 flex justify-end">
        <button
          type="button"
          @click="appStore.closeShortcutsModal()"
          class="px-4 py-1.5 rounded-lg bg-carbon-800 hover:bg-carbon-700 text-carbon-200 text-xs font-semibold cursor-pointer transition-colors"
        >
          Close
        </button>
      </div>
    </template>
  </Dialog>
</template>
