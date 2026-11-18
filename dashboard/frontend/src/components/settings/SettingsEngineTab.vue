<template>
  <div class="space-y-5">
    <!-- Card 1: Job Timeouts & Stale Lock Protection -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Job Timeouts & Stale Lock Protection</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Execution limits ensuring hung sync processes are safely aborted and reclaimed.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Max Job Timeout (Seconds)</label>
          <input
            type="number"
            v-model.number="form.job_timeout_seconds"
            min="60"
            max="86400"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">Default: 1800s (30m). Terminate run if runtime exceeds this threshold.</span>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Stale Lock Timeout (Seconds)</label>
          <input
            type="number"
            v-model.number="form.lock_stale_timeout_seconds"
            min="60"
            max="86400"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">Default: 300s (5m). Reclaim locks older than this duration if process died.</span>
        </div>
      </div>
    </div>

    <!-- Card 2: Rate Limiting & Backoff Multipliers -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Rate Limiting & Backoff Multipliers</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Automated cooldown multipliers to protect against cloud provider API rate limits.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Rate Limit Backoff Period (Hours)</label>
          <input
            type="number"
            v-model.number="form.rate_limit_backoff_hours"
            min="0.1"
            max="48"
            step="0.5"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">Hours to pause sync operations when an upstream 429 is triggered.</span>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Poll Backoff Multiplier</label>
          <input
            type="number"
            v-model.number="form.job_poll_backoff_multiplier"
            min="1.0"
            max="5.0"
            step="0.1"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">Exponential backoff factor for background sync polling loops.</span>
        </div>

        <div class="sm:col-span-2">
          <label class="block text-xs font-semibold text-slate-300 mb-1">Poll Sleep Interval (Initial / Max Seconds)</label>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <input
                type="number"
                v-model.number="form.job_poll_initial_sleep_sec"
                min="0.5"
                step="0.5"
                placeholder="Initial sleep (e.g. 2s)"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
              />
              <span class="text-[10px] text-slate-500 mt-0.5 block">Initial poll sleep interval.</span>
            </div>
            <div>
              <input
                type="number"
                v-model.number="form.job_poll_max_sleep_sec"
                min="1"
                step="1"
                placeholder="Max sleep (e.g. 10s)"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
              />
              <span class="text-[10px] text-slate-500 mt-0.5 block">Maximum backed-off sleep interval.</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Card 3: Daemon Automated Safeguards -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Daemon Automated Safeguards</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Automated self-healing toggles for background daemon engine workers.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
        <label class="flex items-start gap-2.5 cursor-pointer p-3 rounded-lg bg-carbon-canvas border border-carbon-border">
          <input type="checkbox" v-model="form.check_stale_locks" class="rounded bg-carbon-surface border-carbon-border text-electric-blue mt-0.5" />
          <div>
            <span class="text-xs font-semibold text-slate-200 block">Check & Clear Stale Locks</span>
            <span class="text-[10px] text-slate-500">Scan for abandoned PID lockfiles before launching jobs.</span>
          </div>
        </label>

        <label class="flex items-start gap-2.5 cursor-pointer p-3 rounded-lg bg-carbon-canvas border border-carbon-border">
          <input type="checkbox" v-model="form.rate_limit_retry_after_backoff" class="rounded bg-carbon-surface border-carbon-border text-electric-blue mt-0.5" />
          <div>
            <span class="text-xs font-semibold text-slate-200 block">Auto-Retry Jobs Post-Backoff</span>
            <span class="text-[10px] text-slate-500">Automatically re-queue rate-limited jobs when backoff expires.</span>
          </div>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConfigDaemonSettings } from '../../types/models'

const form = defineModel<ConfigDaemonSettings>({ required: true })
</script>
