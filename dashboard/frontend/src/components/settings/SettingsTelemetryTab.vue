<template>
  <div class="space-y-5">
    <!-- Card 1: Telemetry Metrics -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Execution Metrics & Timeseries Ingestion</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Collect duration, speed, and transfer volume metrics for system analytics.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Metrics Collection Engine</label>
          <select
            :value="formTelemetry.enable_metrics ? 'true' : 'false'"
            @change="formTelemetry.enable_metrics = ($event.target as HTMLSelectElement).value === 'true'"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="true">Enabled (Track durations, throughput & counts)</option>
            <option value="false">Disabled</option>
          </select>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Metrics Max In-Memory Records</label>
          <input
            type="number"
            v-model.number="formTelemetry.metrics_max_entries"
            min="100"
            max="20000"
            step="100"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
      </div>
    </div>

    <!-- Card 2: Logging Verbosity & MCP Agent Authority -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Logging Verbosity & MCP Agent Authority</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Control daemon diagnostic verbosity and AI agent write privileges.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Debug Output Verbosity</label>
          <select
            :value="formFeatures.debug_mode ? 'true' : 'false'"
            @change="formFeatures.debug_mode = ($event.target as HTMLSelectElement).value === 'true'"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="false">Standard (Notice, Info, Warnings)</option>
            <option value="true">Verbose Debug Logging (DEBUG_MODE=true)</option>
          </select>
          <span class="text-[10px] text-slate-500 mt-1 block">Verbose mode logs raw Rclone JSON-RPC packets and HTTP access traces.</span>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">MCP Agent Authority</label>
          <select
            :value="formFeatures.mcp_read_only ? 'true' : 'false'"
            @change="formFeatures.mcp_read_only = ($event.target as HTMLSelectElement).value === 'true'"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="false">Read & Write (Full Agent Execution)</option>
            <option value="true">Read Only (Inspection & Safe Dry-Runs Only)</option>
          </select>
          <span class="text-[10px] text-slate-500 mt-1 block">Locks MCP tools against destructive mutation when in Read Only mode.</span>
        </div>
      </div>
    </div>

    <!-- Card 3: Subsystem Feature Toggles -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Subsystem Feature Toggles</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Granular feature flags enabling or bypassing internal sub-modules.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 pt-1">
        <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
          <input type="checkbox" v-model="formFeatures.stale_lock_detection" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
          <div>
            <span class="text-xs font-semibold text-slate-200 block">Stale Lock Detection</span>
            <span class="text-[10px] text-slate-500">Auto-detect crashed process locks.</span>
          </div>
        </label>

        <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
          <input type="checkbox" v-model="formFeatures.rate_limit_backoff" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
          <div>
            <span class="text-xs font-semibold text-slate-200 block">Rate Limit Backoff</span>
            <span class="text-[10px] text-slate-500">Provider throttling protection.</span>
          </div>
        </label>

        <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
          <input type="checkbox" v-model="formFeatures.daemon_health_check" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
          <div>
            <span class="text-xs font-semibold text-slate-200 block">Daemon Health Checks</span>
            <span class="text-[10px] text-slate-500">Periodic Rclone RC socket probes.</span>
          </div>
        </label>

        <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
          <input type="checkbox" v-model="formFeatures.smart_notifications" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
          <div>
            <span class="text-xs font-semibold text-slate-200 block">Smart Alert Throttling</span>
            <span class="text-[10px] text-slate-500">Suppress repetitive error spam.</span>
          </div>
        </label>

        <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border sm:col-span-2">
          <input type="checkbox" v-model="formFeatures.webhook_alerts" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
          <div>
            <span class="text-xs font-semibold text-slate-200 block">External Webhook Alerts</span>
            <span class="text-[10px] text-slate-500">Dispatch sync failure alerts to configured webhook endpoint.</span>
          </div>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConfigTelemetrySettings, ConfigFeatureSettings } from '../../types/models'

const formTelemetry = defineModel<ConfigTelemetrySettings>('formTelemetry', { required: true })
const formFeatures = defineModel<ConfigFeatureSettings>('formFeatures', { required: true })
</script>
