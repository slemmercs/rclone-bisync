<script setup lang="ts">
import { computed } from 'vue'
import type { SyncFSMState } from '../types/fsm'

const props = withDefaults(
  defineProps<{
    state: SyncFSMState
    reason?: string
    pulse?: boolean
  }>(),
  {
    pulse: false,
  }
)

const emit = defineEmits<{
  (e: 'click'): void
}>()

const badgeConfig = computed(() => {
  switch (props.state) {
    case 'RUNNING':
      return {
        label: 'RUNNING',
        icon: 'pi pi-spin pi-spinner',
        classes: 'bg-electric-950/80 text-electric-400 border border-electric-600/60 shadow-[0_0_12px_rgba(14,165,233,0.25)]',
        dot: 'bg-electric-400 animate-ping',
      }
    case 'QUEUED':
      return {
        label: 'QUEUED',
        icon: 'pi pi-clock',
        classes: 'bg-carbon-800 text-electric-300 border border-carbon-600',
        dot: 'bg-electric-300 animate-pulse',
      }
    case 'REVIEW_REQUIRED':
      return {
        label: 'REVIEW REQUIRED',
        icon: 'pi pi-exclamation-triangle',
        classes: 'bg-amber-950/80 text-amber-400 border border-amber-600/70 shadow-[0_0_10px_rgba(245,158,11,0.2)]',
        dot: 'bg-amber-400 animate-pulse',
      }
    case 'RESOLVING':
      return {
        label: 'RESOLVING',
        icon: 'pi pi-cog pi-spin',
        classes: 'bg-indigo-950/80 text-indigo-300 border border-indigo-600/60',
        dot: 'bg-indigo-400',
      }
    case 'COMPLETED':
      return {
        label: 'COMPLETED',
        icon: 'pi pi-check-circle',
        classes: 'bg-emerald-950/80 text-emerald-400 border border-emerald-600/50',
        dot: 'bg-emerald-400',
      }
    case 'ABORTED':
      return {
        label: 'ABORTED',
        icon: 'pi pi-times-circle',
        classes: 'bg-red-950/80 text-red-400 border border-red-600/60 shadow-[0_0_8px_rgba(239,68,68,0.2)]',
        dot: 'bg-red-400',
      }
    case 'IDLE':
    default:
      return {
        label: 'IDLE',
        icon: 'pi pi-circle',
        classes: 'bg-carbon-850 text-carbon-400 border border-carbon-700',
        dot: 'bg-carbon-500',
      }
  }
})
</script>

<template>
  <div
    class="inline-flex items-center gap-2 px-2.5 py-1 rounded-full text-xs font-mono font-medium tracking-wide transition-all select-none"
    :class="[
      badgeConfig.classes,
      props.state === 'REVIEW_REQUIRED' ? 'cursor-pointer hover:brightness-125 active:scale-95' : ''
    ]"
    :title="reason ? `${badgeConfig.label}: ${reason}` : (props.state === 'REVIEW_REQUIRED' ? 'Click to open Conflict Review' : badgeConfig.label)"
    @click="emit('click')"
  >
    <span class="relative flex h-2 w-2">
      <span v-if="props.state === 'RUNNING' || props.state === 'QUEUED'" class="absolute inline-flex h-full w-full rounded-full opacity-75" :class="badgeConfig.dot"></span>
      <span class="relative inline-flex rounded-full h-2 w-2" :class="badgeConfig.dot"></span>
    </span>
    <i :class="badgeConfig.icon" class="text-[11px]"></i>
    <span>{{ badgeConfig.label }}</span>
  </div>
</template>
