<template>
  <div class="space-y-5">
    <!-- Card 1: Log Rotation & Retention -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Log Rotation & Archival Retention</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Bounded disk consumption rules and archive retention count for <code class="text-slate-300">bisync.log</code>.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Max Log Size (MB)</label>
          <input
            type="number"
            v-model.number="formLogging.max_log_size_mb"
            min="1"
            max="500"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">Rotates log file when this size threshold is reached.</span>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Log Backup Retention Count</label>
          <input
            type="number"
            v-model.number="formLogging.log_backup_count"
            min="1"
            max="50"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">Number of archived log backups to retain on disk.</span>
        </div>
      </div>
    </div>

    <!-- Card 2: Post-Sync Cleanup -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Post-Sync Ephemeral File Cleanup</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Comma-separated filename patterns automatically pruned after successful sync completion.</p>
      </div>
      <div>
        <label class="block text-xs font-semibold text-slate-300 mb-1">Cleanup Patterns (Globs)</label>
        <input
          type="text"
          v-model="formCleanup.files_to_cleanup"
          placeholder="*.tmp, *.temp, *.conflict, *.~lock.*"
          class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
        />
        <span class="text-[10px] text-slate-500 mt-1 block">Matches are safely removed from the synced root after each completed run.</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConfigLoggingSettings, ConfigCleanupSettings } from '../../types/models'

const formLogging = defineModel<ConfigLoggingSettings>('formLogging', { required: true })
const formCleanup = defineModel<ConfigCleanupSettings>('formCleanup', { required: true })
</script>
