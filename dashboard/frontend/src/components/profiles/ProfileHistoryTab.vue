<script setup lang="ts">
import type { Profile, ProfileVersion } from '../../types/models'

defineProps<{
  profile: Partial<Profile>
  records: ProfileVersion[]
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'rollback', version: number): void
}>()
</script>

<template>
  <div class="space-y-3 max-h-[60vh] overflow-y-auto pr-1">
    <div class="flex items-center justify-between p-3 rounded-xl bg-carbon-bg border border-carbon-border">
      <div>
        <div class="text-sm font-bold text-white">{{ profile.name }}</div>
        <div class="text-xs font-mono text-slate-400">
          ID: {{ profile.id }} • Current Version: v{{ profile.version || 1 }}
        </div>
      </div>
      <button
        @click="emit('back')"
        class="px-3 py-1.5 bg-carbon-elevated hover:bg-carbon-border text-slate-300 rounded-lg text-xs transition cursor-pointer"
      >
        Back to Profiles
      </button>
    </div>

    <div v-if="loading" class="p-8 text-center text-electric-blue text-xs animate-pulse">
      Loading historical snapshots...
    </div>
    <div
      v-else-if="records.length === 0"
      class="p-8 text-center text-slate-500 text-xs border border-dashed border-carbon-border rounded-xl"
    >
      No version history recorded yet. Historical snapshots are archived each time profile settings are modified.
    </div>
    <div v-else class="space-y-2">
      <div
        v-for="rec in records"
        :key="rec.version"
        class="p-3 rounded-xl bg-carbon-bg border border-carbon-border flex items-center justify-between gap-3 text-xs"
      >
        <div class="space-y-0.5 min-w-0">
          <div class="flex items-center gap-2">
            <span class="font-mono font-bold text-indigo-400">v{{ rec.version }}</span>
            <span class="text-slate-500">•</span>
            <span class="text-slate-300 font-medium">{{ rec.reason || 'Configuration update' }}</span>
            <span
              v-if="rec.version === profile.version"
              class="text-[9px] px-1.5 py-0.2 rounded font-bold uppercase bg-emerald-500/20 text-emerald-300 border border-emerald-500/30"
            >
              Current
            </span>
          </div>
          <div class="text-[11px] text-slate-400 font-mono flex items-center gap-2">
            <span>{{ new Date(rec.timestamp).toLocaleString() }}</span>
            <span>•</span>
            <span class="text-slate-500">{{ rec.snapshot.remote }} &harr; {{ rec.snapshot.local_path }}</span>
          </div>
        </div>
        <button
          v-if="rec.version !== profile.version"
          type="button"
          @click="emit('rollback', rec.version)"
          class="px-3 py-1 bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 rounded-lg text-xs font-semibold border border-amber-500/30 transition cursor-pointer shrink-0"
        >
          Rollback to v{{ rec.version }}
        </button>
      </div>
    </div>
  </div>
</template>
