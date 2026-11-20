<template>
  <div class="space-y-5">
    <!-- Card 1: Ceiling & Presets -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div class="flex items-center justify-between">
        <div>
          <h4 class="text-xs font-bold text-white">Dynamic Throttling Ceiling</h4>
          <p class="text-[11px] text-slate-400 mt-0.5">
            Active upload/download bandwidth cap managed via Rclone RC <code class="text-slate-300">core/bwlimit</code>.
          </p>
        </div>
        <span
          class="px-2.5 py-1 rounded-lg text-xs font-mono font-bold uppercase tracking-wider border"
          :class="bandwidthStore.currentLimit !== 'off' && bandwidthStore.currentLimit !== ''
            ? 'bg-amber-500/20 text-amber-300 border-amber-500/30'
            : 'bg-carbon-elevated text-slate-300 border-carbon-border'"
        >
          {{ bandwidthStore.currentLimit !== 'off' && bandwidthStore.currentLimit !== '' ? ('Limit: ' + bandwidthStore.rateFormatted) : 'Unlimited (off)' }}
        </span>
      </div>

      <div class="pt-2 border-t border-carbon-border">
        <label class="block text-xs font-semibold text-slate-300 mb-2">Throttling Presets</label>
        <div class="grid grid-cols-5 gap-2">
          <button
            v-for="p in bandwidthStore.presets"
            :key="p"
            type="button"
            @click="handleSetPreset(p)"
            :disabled="bandwidthStore.updating"
            class="py-1.5 px-2 rounded-lg text-xs font-mono font-semibold transition-all border text-center cursor-pointer"
            :class="(bandwidthStore.currentLimit === p || (p === 'off' && bandwidthStore.currentLimit === ''))
              ? 'bg-electric-blue text-black border-sky-400 font-extrabold shadow-sm'
              : 'bg-carbon-canvas hover:bg-carbon-surface text-slate-300 border-carbon-border'"
          >
            {{ p === 'off' ? 'Max (off)' : p }}
          </button>
        </div>
      </div>

      <div class="pt-2 border-t border-carbon-border">
        <label class="block text-xs font-semibold text-slate-300 mb-1.5">Custom Rate Limit</label>
        <form @submit.prevent="handleApplyCustom" class="flex items-center gap-2">
          <input
            type="text"
            v-model="bandwidthStore.customInput"
            placeholder="e.g. 15M, 500k, 1.5G, off"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <button
            type="submit"
            :disabled="bandwidthStore.updating || !bandwidthStore.customInput.trim()"
            class="px-4 py-1.5 bg-electric-blue hover:bg-sky-400 text-black rounded-lg text-xs font-bold shrink-0 transition-colors cursor-pointer disabled:opacity-40"
          >
            {{ bandwidthStore.updating ? 'Applying...' : 'Apply Limit' }}
          </button>
        </form>
      </div>
    </div>

    <!-- Card 2: Live Engine Telemetry -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <h4 class="text-xs font-bold text-white">Live Engine Transfer Telemetry</h4>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs font-mono">
        <div class="bg-carbon-canvas p-2.5 rounded-xl border border-carbon-border">
          <div class="text-[9px] uppercase font-bold text-slate-500">Live Speed</div>
          <div class="text-sm font-bold text-white mt-0.5">{{ formatSpeed(bandwidthStore.speed) }}</div>
        </div>
        <div class="bg-carbon-canvas p-2.5 rounded-xl border border-carbon-border">
          <div class="text-[9px] uppercase font-bold text-slate-500">Transferred</div>
          <div class="text-sm font-bold text-electric-blue mt-0.5">{{ formatBytes(bandwidthStore.bytes) }}</div>
        </div>
        <div class="bg-carbon-canvas p-2.5 rounded-xl border border-carbon-border">
          <div class="text-[9px] uppercase font-bold text-slate-500">Transfers</div>
          <div class="text-sm font-bold text-emerald-400 mt-0.5">{{ bandwidthStore.transfers }}</div>
        </div>
        <div class="bg-carbon-canvas p-2.5 rounded-xl border border-carbon-border">
          <div class="text-[9px] uppercase font-bold text-slate-500">Errors</div>
          <div class="text-sm font-bold mt-0.5" :class="bandwidthStore.errors > 0 ? 'text-rose-400' : 'text-slate-300'">
            {{ bandwidthStore.errors }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useBandwidthStore } from '../../stores/bandwidth'
import { useToastNotification } from '../../composables/useToastNotification'
import { formatBytes, formatSpeed } from '../../composables/useFormatters'

const bandwidthStore = useBandwidthStore()
const toast = useToastNotification()

onMounted(() => {
  bandwidthStore.fetchBandwidthLimit()
  bandwidthStore.fetchCoreStats()
})

async function handleSetPreset(rate: string) {
  try {
    await bandwidthStore.setBandwidthLimit(rate)
    const label = rate === 'off' ? 'Bandwidth throttle disabled (unlimited speed)' : `Bandwidth limit set to ${rate}`
    toast.success(label, 'Bandwidth Limit')
  } catch (err: any) {
    toast.error(err?.message || 'Failed to set bandwidth limit', 'Bandwidth Error')
  }
}

async function handleApplyCustom() {
  const rate = bandwidthStore.customInput.trim()
  if (!rate) return
  try {
    await bandwidthStore.applyCustomBandwidthLimit()
    toast.success(`Bandwidth limit set to ${rate}`, 'Bandwidth Limit')
  } catch (err: any) {
    toast.error(err?.message || 'Failed to set bandwidth limit', 'Bandwidth Error')
  }
}
</script>
