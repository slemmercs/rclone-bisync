<template>
  <div class="space-y-5">
    <!-- Card 1: Bootstrap Target Fallbacks -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Bootstrap Target Fallbacks</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">
          Default paths configured in <span class="text-electric-blue font-mono">bisync.env</span> (<code class="text-slate-300">LOCAL_PATH</code> & <code class="text-slate-300">REMOTE_PATH</code>). Used as bootstrap defaults for manual CLI bisync runs.
          <span class="text-amber-300 block sm:inline">Active sync profiles manage their own isolated path mappings in Profiles.</span>
        </p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Fallback Local Bootstrap Path</label>
          <input
            type="text"
            v-model="form.default_local_path"
            placeholder="~/sync_folder"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Fallback Remote Bootstrap Target</label>
          <input
            type="text"
            v-model="form.default_remote_path"
            placeholder="drive:backup"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
      </div>
    </div>

    <!-- Card 2: Profile Baseline Safeguards -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Sync Profile Baseline Safeguards</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Default safety and conflict heuristics applied when creating new sync profiles.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Default Max Delete %</label>
          <input
            type="number"
            v-model.number="form.max_delete"
            min="0"
            max="100"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">0 = Abort on any deletion; 1-100 = allowable % threshold.</span>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Conflict Resolution Mode</label>
          <select
            v-model="form.conflict_resolve"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="newer">newer (Keep newest file)</option>
            <option value="older">older (Keep older file)</option>
            <option value="larger">larger (Keep larger file)</option>
            <option value="smaller">smaller (Keep smaller file)</option>
            <option value="path1">path1 (Prefer Local)</option>
            <option value="path2">path2 (Prefer Remote)</option>
          </select>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Comparison Mode</label>
          <select
            v-model="form.compare_mode"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="modtime">modtime</option>
            <option value="size,modtime">size,modtime (Standard)</option>
            <option value="size,modtime,checksum">size,modtime,checksum (Thorough)</option>
            <option value="checksum">checksum (Hash only)</option>
          </select>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Canary Token Validation</label>
          <select
            :value="form.canary_validation ? 'true' : 'false'"
            @change="form.canary_validation = ($event.target as HTMLSelectElement).value === 'true'"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="true">Enabled (Strict RCLONE_TEST Guard)</option>
            <option value="false">Disabled (Bypass Probing)</option>
          </select>
        </div>
      </div>
    </div>
    <!-- Card 3: Pruning & Explorer Scoping -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div>
        <h4 class="text-xs font-bold text-white">Pruning & Explorer Scoping</h4>
        <p class="text-[11px] text-slate-400 mt-0.5">Directory pruning behavior and dual explorer workspace confinement.</p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1.5">Empty Directory Cleanup</label>
          <div class="space-y-2">
            <label class="flex items-center gap-2 cursor-pointer p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
              <input type="checkbox" v-model="form.remove_empty_dirs" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
              <span class="text-xs text-slate-300">Remove Empty Dirs</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
              <input type="checkbox" v-model="form.create_empty_dirs" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
              <span class="text-xs text-slate-300">Create Empty Dirs</span>
            </label>
          </div>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Explorer Remote Scoping</label>
          <select
            :value="form.lock_explorer_remote ? 'true' : 'false'"
            @change="form.lock_explorer_remote = ($event.target as HTMLSelectElement).value === 'true'"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="false">Unlocked (Freely select any remote)</option>
            <option value="true">Locked to Active Profile Remote</option>
          </select>
          <span class="text-[10px] text-slate-500 mt-1 block">Locks dual explorer right pane strictly to active profile remote.</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GlobalDefaults } from '../../types/models'

const form = defineModel<GlobalDefaults>({ required: true })
</script>
