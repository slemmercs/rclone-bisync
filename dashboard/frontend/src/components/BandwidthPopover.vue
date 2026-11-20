<template>
  <Popover
    ref="popoverRef"
    :dismissable="true"
    @hide="onHide"
    @show="onShow"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border shadow-2xl rounded-2xl w-80 sm:w-96 p-4 text-xs font-mono text-slate-300 z-50' },
      content: { class: 'p-0 space-y-3' }
    }"
  >
    <!-- Header -->
    <div class="flex items-center justify-between pb-2 border-b border-carbon-border">
      <div class="flex items-center gap-2">
        <i class="pi pi-gauge text-electric-blue text-sm"></i>
        <span class="font-bold text-white text-xs">Bandwidth Throttling</span>
      </div>
      <span
        class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border"
        :class="bandwidthStore.currentLimit !== 'off' && bandwidthStore.currentLimit !== ''
          ? 'bg-amber-500/20 text-amber-300 border-amber-500/30'
          : 'bg-carbon-elevated text-slate-300 border-carbon-border'"
      >
        {{ bandwidthStore.currentLimit !== 'off' && bandwidthStore.currentLimit !== '' ? bandwidthStore.rateFormatted : 'Unlimited (off)' }}
      </span>
    </div>

    <!-- Presets -->
    <div>
      <label class="block text-[11px] font-semibold text-slate-300 mb-1.5">Presets</label>
      <div class="grid grid-cols-5 gap-1.5">
        <button
          v-for="p in bandwidthStore.presets"
          :key="p"
          type="button"
          @click="handleSetPreset(p)"
          :disabled="bandwidthStore.updating"
          class="py-1 px-1 rounded-lg text-[10px] font-mono font-bold transition-all border text-center cursor-pointer"
          :class="(bandwidthStore.currentLimit === p || (p === 'off' && bandwidthStore.currentLimit === ''))
            ? 'bg-electric-blue text-black border-sky-400 font-extrabold shadow-sm'
            : 'bg-carbon-elevated hover:bg-carbon-surface text-slate-300 border-carbon-border hover:border-slate-600'"
        >
          {{ p === 'off' ? 'Max (off)' : p }}
        </button>
      </div>
    </div>

    <!-- Custom Rate Form -->
    <div>
      <label class="block text-[11px] font-semibold text-slate-300 mb-1">Custom Ceiling</label>
      <form @submit.prevent="handleApplyCustom" class="flex items-center gap-1.5">
        <input
          type="text"
          v-model="bandwidthStore.customInput"
          placeholder="e.g. 15M, 500k, off"
          class="flex-1 bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1 text-xs text-white font-mono placeholder:text-slate-600 focus:outline-none focus:border-electric-blue"
        />
        <button
          type="submit"
          :disabled="bandwidthStore.updating || !bandwidthStore.customInput.trim()"
          class="px-3 py-1 bg-electric-blue hover:bg-sky-400 disabled:opacity-40 disabled:cursor-not-allowed text-black font-bold rounded-lg text-xs shrink-0 transition-colors cursor-pointer"
        >
          {{ bandwidthStore.updating ? '...' : 'Apply' }}
        </button>
      </form>
    </div>

    <!-- Live Transfer Telemetry Grid -->
    <div class="grid grid-cols-4 gap-1.5 pt-1 text-center font-mono">
      <div class="bg-carbon-canvas p-1.5 rounded-lg border border-carbon-border">
        <div class="text-[8px] uppercase text-slate-500 font-bold">Speed</div>
        <div class="text-[11px] font-bold text-white mt-0.5 truncate">{{ formatSpeed(bandwidthStore.speed) }}</div>
      </div>
      <div class="bg-carbon-canvas p-1.5 rounded-lg border border-carbon-border">
        <div class="text-[8px] uppercase text-slate-500 font-bold">Total</div>
        <div class="text-[11px] font-bold text-electric-blue mt-0.5 truncate">{{ formatBytes(bandwidthStore.bytes) }}</div>
      </div>
      <div class="bg-carbon-canvas p-1.5 rounded-lg border border-carbon-border">
        <div class="text-[8px] uppercase text-slate-500 font-bold">Transfers</div>
        <div class="text-[11px] font-bold text-emerald-400 mt-0.5">{{ bandwidthStore.transfers }}</div>
      </div>
      <div class="bg-carbon-canvas p-1.5 rounded-lg border border-carbon-border">
        <div class="text-[8px] uppercase text-slate-500 font-bold">Errors</div>
        <div class="text-[11px] font-bold mt-0.5" :class="bandwidthStore.errors > 0 ? 'text-rose-400' : 'text-slate-400'">{{ bandwidthStore.errors }}</div>
      </div>
    </div>
  </Popover>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import Popover from 'primevue/popover'
import { useBandwidthStore } from '../stores/bandwidth'
import { useToastNotification } from '../composables/useToastNotification'
import { formatBytes, formatSpeed } from '../composables/useFormatters'

const bandwidthStore = useBandwidthStore()
const toast = useToastNotification()
const popoverRef = ref<InstanceType<typeof Popover> | null>(null)

function toggle(event: Event) {
  popoverRef.value?.toggle(event)
}

function show(event: Event, target?: any) {
  popoverRef.value?.show(event, target)
}

function hide() {
  popoverRef.value?.hide()
}

function onShow() {
  bandwidthStore.openPopover()
}

function onHide() {
  bandwidthStore.closePopover()
}

watch(() => bandwidthStore.popoverOpen, (isOpen) => {
  if (!isOpen) popoverRef.value?.hide()
})

async function handleSetPreset(rate: string) {
  try {
    await bandwidthStore.setBandwidthLimit(rate)
    const label = rate === 'off' ? 'Throttling disabled (unlimited speed)' : `Bandwidth ceiling set to ${rate}`
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
    toast.success(`Bandwidth ceiling set to ${rate}`, 'Bandwidth Limit')
  } catch (err: any) {
    toast.error(err?.message || 'Failed to set bandwidth limit', 'Bandwidth Error')
  }
}

defineExpose({ toggle, show, hide })
</script>
