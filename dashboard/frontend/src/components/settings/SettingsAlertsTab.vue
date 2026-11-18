<template>
  <div class="space-y-5">
    <!-- Card 1: Notification Engine -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Desktop & System Notification Engine</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Desktop notification parameters and throttle intervals.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3.5 pt-1">
        <div class="sm:col-span-1">
          <label class="block text-xs font-semibold text-slate-300 mb-1">Application Name</label>
          <input
            type="text"
            v-model="form.app_name"
            placeholder="Rclone Bisync"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Success Alert (min)</label>
          <input
            type="number"
            v-model.number="form.success_interval_min"
            min="0"
            max="1440"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Error Repeat (hrs)</label>
          <input
            type="number"
            v-model.number="form.error_repeat_hours"
            min="0.1"
            max="72"
            step="0.5"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
      </div>
    </div>

    <!-- Card 2: External Endpoints -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">External Integration Endpoints</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Webhooks for operational alerts and external uptime monitoring.</p>
      </div>
      <div class="space-y-3 pt-1">
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="text-xs font-semibold text-slate-300">Alert Webhook URL</label>
            <span class="text-[10px] text-slate-500 font-mono">Timeout: {{ form.webhook_timeout_sec || 5 }}s</span>
          </div>
          <input
            type="url"
            v-model="form.webhook_url"
            placeholder="https://hooks.slack.com/services/... or https://discord.com/api/webhooks/..."
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">POST JSON payload dispatched on sync failure, canary trip, or excessive deletion.</span>
        </div>

        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="text-xs font-semibold text-slate-300">Watchdog Heartbeat Ping URL</label>
            <span class="text-[10px] text-slate-500 font-mono">Timeout: {{ form.watchdog_timeout_sec || 5 }}s</span>
          </div>
          <input
            type="url"
            v-model="form.watchdog_ping_url"
            placeholder="https://nosnch.in/... or https://hc-ping.com/..."
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">GET ping dispatched following every successfully completed scheduled cycle.</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConfigNotificationSettings } from '../../types/models'

const form = defineModel<ConfigNotificationSettings>({ required: true })
</script>
