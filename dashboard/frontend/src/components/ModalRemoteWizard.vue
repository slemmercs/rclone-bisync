<template>
  <Dialog
    :visible="dedupeStore.wizardOpen"
    @update:visible="val => !val && dedupeStore.closeWizard()"
    modal
    :style="{ width: '90vw', maxWidth: '850px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl shadow-2xl overflow-hidden' },
      header: { class: 'p-4 sm:p-5 border-b border-carbon-border bg-carbon-surface flex justify-between items-center text-white' },
      content: { class: 'p-5 bg-carbon-canvas flex flex-col max-h-[80vh] overflow-y-auto space-y-4' },
      closeButton: { class: 'text-slate-400 hover:text-white hover:bg-carbon-elevated rounded-lg p-1.5 transition-colors cursor-pointer' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-xl bg-electric-blue/10 border border-electric-blue/30 flex items-center justify-center text-electric-blue shrink-0">
          <i class="pi pi-plus text-xs"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white uppercase tracking-wider">Remote Creation Wizard</h3>
          <div class="text-[11px] text-slate-400">Guided cloud remote configuration and sync profile bridging</div>
        </div>
      </div>
    </template>

    <!-- Stepper Indicator -->
    <div class="grid grid-cols-3 gap-2 border-b border-carbon-border/80 pb-3 shrink-0 text-xs font-mono">
      <button
        type="button"
        @click="step = 1"
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg border transition-all cursor-pointer text-left"
        :class="step === 1 ? 'bg-electric-blue/20 border-electric-blue text-electric-blue font-bold' : (step > 1 ? 'bg-carbon-surface border-carbon-border text-emerald-400' : 'bg-carbon-canvas border-carbon-border text-slate-500')"
      >
        <span class="w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold" :class="step === 1 ? 'bg-electric-blue text-white' : (step > 1 ? 'bg-emerald-500 text-black' : 'bg-carbon-elevated text-slate-400')">
          {{ step > 1 ? '✓' : '1' }}
        </span>
        <span class="truncate">Select Provider</span>
      </button>

      <button
        type="button"
        @click="selectedProvider && (step = 2)"
        :disabled="!selectedProvider"
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg border transition-all text-left"
        :class="step === 2 ? 'bg-electric-blue/20 border-electric-blue text-electric-blue font-bold cursor-pointer' : (step > 2 ? 'bg-carbon-surface border-carbon-border text-emerald-400 cursor-pointer' : (selectedProvider ? 'bg-carbon-canvas border-carbon-border text-slate-400 cursor-pointer' : 'bg-carbon-canvas border-carbon-border text-slate-600 cursor-not-allowed'))"
      >
        <span class="w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold" :class="step === 2 ? 'bg-electric-blue text-white' : (step > 2 ? 'bg-emerald-500 text-black' : 'bg-carbon-elevated text-slate-400')">
          {{ step > 2 ? '✓' : '2' }}
        </span>
        <span class="truncate">Configure & Bridge</span>
      </button>

      <button
        type="button"
        @click="canProceedToStep3 && (step = 3)"
        :disabled="!canProceedToStep3"
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg border transition-all text-left"
        :class="step === 3 ? 'bg-electric-blue/20 border-electric-blue text-electric-blue font-bold cursor-pointer' : (canProceedToStep3 ? 'bg-carbon-canvas border-carbon-border text-slate-400 cursor-pointer' : 'bg-carbon-canvas border-carbon-border text-slate-600 cursor-not-allowed')"
      >
        <span class="w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold" :class="step === 3 ? 'bg-electric-blue text-white' : 'bg-carbon-elevated text-slate-400'">
          3
        </span>
        <span class="truncate">Review & Deploy</span>
      </button>
    </div>

    <!-- STEP 1: Select Provider -->
    <div v-if="step === 1" class="space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-1.5 text-xs font-mono">
          <button
            v-for="cat in (['popular', 'cloud', 'object', 'protocol', 'all'] as const)"
            :key="cat"
            type="button"
            @click="category = cat"
            :class="category === cat ? 'bg-electric-blue text-white font-bold' : 'bg-carbon-elevated text-slate-300 hover:text-white'"
            class="px-2.5 py-1 rounded-lg transition-colors cursor-pointer capitalize"
          >
            {{ cat === 'object' ? 'Object (S3)' : cat }}
          </button>
        </div>
        <div class="relative w-full sm:w-60">
          <input
            type="text"
            v-model="search"
            placeholder="Search providers..."
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-electric-blue font-mono"
          />
          <i class="pi pi-search text-slate-500 absolute left-2.5 top-2.5 text-xs pointer-events-none"></i>
        </div>
      </div>

      <div v-if="dedupeStore.providersLoading" class="py-16 text-center text-slate-400 space-y-2">
        <i class="pi pi-spin pi-spinner text-xl text-electric-blue"></i>
        <div class="text-xs font-mono">Fetching provider catalog from Rclone...</div>
      </div>

      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <div
          v-for="p in filteredProviders"
          :key="p.name"
          @click="selectProvider(p)"
          class="p-3.5 rounded-xl border bg-carbon-surface/60 hover:bg-carbon-elevated/80 transition-all cursor-pointer flex flex-col justify-between group space-y-2"
          :class="selectedProvider?.name === p.name ? 'border-electric-blue ring-1 ring-electric-blue' : 'border-carbon-border hover:border-carbon-border/80'"
        >
          <div class="space-y-1.5">
            <div class="flex items-center justify-between gap-2">
              <span class="font-bold text-sm text-white group-hover:text-electric-300 transition-colors">{{ p.name }}</span>
              <div class="flex items-center gap-1">
                <span v-if="p.category === 'popular'" class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-electric-blue/10 text-electric-blue border border-electric-blue/20">Popular</span>
                <span v-if="p.oauth_supported" class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">OAuth</span>
              </div>
            </div>
            <p class="text-xs text-slate-400 line-clamp-2">{{ p.description || p.name }}</p>
          </div>
          <div class="flex items-center justify-between text-[11px] text-slate-500 font-mono pt-2 border-t border-carbon-border">
            <span>{{ p.prefix || p.name }}</span>
            <span class="text-electric-blue group-hover:translate-x-0.5 transition-transform">Configure →</span>
          </div>
        </div>
      </div>
    </div>

    <!-- STEP 2: Configure & Bridge -->
    <div v-else-if="step === 2" class="space-y-4">
      <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="text-xs text-slate-400">Selected Provider:</span>
          <span class="text-xs font-bold text-electric-blue uppercase font-mono">{{ selectedProvider?.name }}</span>
        </div>
        <button type="button" @click="step = 1" class="text-xs text-slate-400 hover:text-white cursor-pointer font-mono">
          Change Provider
        </button>
      </div>

      <div class="space-y-3 font-mono text-xs">
        <div>
          <label class="block text-slate-300 font-semibold mb-1">Remote Name *</label>
          <input
            type="text"
            v-model="remoteName"
            placeholder="e.g. gdrive_backup"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-white focus:outline-none focus:border-electric-blue"
          />
        </div>

        <!-- Sync Profile Bridge Toggle -->
        <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
          <label class="flex items-center gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              v-model="createProfile"
              class="rounded bg-carbon-canvas border-carbon-border text-electric-blue focus:ring-0 w-3.5 h-3.5"
            />
            <span class="text-xs text-white font-semibold">Automatically provision linked sync profile</span>
          </label>

          <div v-if="createProfile" class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 border-t border-carbon-border/60">
            <div>
              <label class="block text-slate-400 mb-1 text-[11px]">Local Directory Path *</label>
              <input
                type="text"
                v-model="localPath"
                placeholder="e.g. /home/user/drive_sync"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200"
              />
            </div>
            <div>
              <label class="block text-slate-400 mb-1 text-[11px]">Remote Subpath</label>
              <input
                type="text"
                v-model="remoteSubpath"
                placeholder="e.g. backup/vault (optional)"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200"
              />
            </div>
            <div>
              <label class="block text-slate-400 mb-1 text-[11px]">Sync Schedule</label>
              <select
                v-model="schedule"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 cursor-pointer"
              >
                <option value="manual">Manual (No Schedule)</option>
                <option value="15m">Every 15 Minutes</option>
                <option value="30m">Every 30 Minutes</option>
                <option value="1h">Hourly</option>
                <option value="4h">Every 4 Hours</option>
                <option value="daily">Daily</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- STEP 3: Review & Deploy -->
    <div v-else-if="step === 3" class="space-y-4 font-mono text-xs">
      <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <h4 class="font-bold text-white text-sm">Deployment Review</h4>
        <div class="grid grid-cols-2 gap-2 text-slate-300">
          <div><span class="text-slate-500">Remote Name:</span> {{ remoteName }}</div>
          <div><span class="text-slate-500">Provider:</span> {{ selectedProvider?.name }}</div>
          <div><span class="text-slate-500">Linked Profile:</span> {{ createProfile ? 'Enabled' : 'Disabled' }}</div>
          <div v-if="createProfile"><span class="text-slate-500">Local Path:</span> {{ localPath || '--' }}</div>
          <div v-if="createProfile"><span class="text-slate-500">Schedule:</span> {{ schedule }}</div>
        </div>
      </div>

      <div v-if="errorMsg" class="p-3 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl">
        {{ errorMsg }}
      </div>
    </div>

    <!-- Footer Controls -->
    <div class="flex items-center justify-between pt-3 border-t border-carbon-border shrink-0">
      <button
        type="button"
        @click="step > 1 ? step-- : dedupeStore.closeWizard()"
        class="px-3.5 py-1.5 rounded-lg bg-carbon-elevated hover:bg-slate-700 text-slate-300 text-xs font-mono border border-carbon-border cursor-pointer transition-colors"
      >
        {{ step > 1 ? '← Back' : 'Cancel' }}
      </button>

      <div class="flex items-center gap-2">
        <button
          v-if="step < 3"
          type="button"
          @click="step === 1 ? (selectedProvider && (step = 2)) : (canProceedToStep3 && (step = 3))"
          :disabled="step === 1 ? !selectedProvider : !canProceedToStep3"
          class="px-4 py-1.5 rounded-lg bg-electric-blue hover:bg-sky-400 text-black font-bold text-xs font-mono cursor-pointer transition-colors disabled:opacity-40"
        >
          Next Step →
        </button>
        <button
          v-else
          type="button"
          @click="handleDeploy()"
          :disabled="submitting"
          class="px-4 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-black font-bold text-xs font-mono cursor-pointer transition-colors disabled:opacity-40 flex items-center gap-1.5"
        >
          <i class="pi" :class="submitting ? 'pi-spin pi-spinner' : 'pi-check'"></i>
          <span>Deploy Remote</span>
        </button>
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import { useDedupeStore } from '../stores/dedupe'
import { useRemotesStore } from '../stores/remotes'
import { useProfilesStore } from '../stores/profiles'
import type { ProviderCatalogItem, CreateRemotePayload } from '../types/dedupe'

const dedupeStore = useDedupeStore()
const remotesStore = useRemotesStore()
const profilesStore = useProfilesStore()

const step = ref<number>(1)
const category = ref<'popular' | 'cloud' | 'object' | 'protocol' | 'all'>('popular')
const search = ref<string>('')
const selectedProvider = ref<ProviderCatalogItem | null>(null)

const remoteName = ref<string>('')
const parameters = ref<Record<string, any>>({})
const createProfile = ref<boolean>(true)
const localPath = ref<string>('')
const remoteSubpath = ref<string>('')
const schedule = ref<string>('manual')
const errorMsg = ref<string | null>(null)
const submitting = ref<boolean>(false)

const canProceedToStep3 = computed(() => !!selectedProvider.value && !!remoteName.value.trim())

const filteredProviders = computed(() => {
  let list = dedupeStore.providers || []
  if (category.value === 'popular') {
    list = list.filter(p => p.category === 'popular' || ['drive', 'onedrive', 's3', 'b2', 'dropbox'].includes(p.name))
  } else if (category.value !== 'all') {
    list = list.filter(p => p.category === category.value)
  }

  const q = search.value.toLowerCase().trim()
  if (q) {
    list = list.filter(p => (p.name || '').toLowerCase().includes(q) || (p.description || '').toLowerCase().includes(q))
  }
  return list
})

function selectProvider(p: ProviderCatalogItem) {
  selectedProvider.value = p
  if (!remoteName.value) {
    remoteName.value = `${p.name}_remote`
  }
  step.value = 2
}

async function handleDeploy() {
  if (!selectedProvider.value || !remoteName.value.trim()) return
  submitting.value = true
  errorMsg.value = null

  const payload: CreateRemotePayload = {
    name: remoteName.value.trim(),
    type: selectedProvider.value.name,
    parameters: parameters.value,
    create_profile: createProfile.value,
    local_path: localPath.value.trim() || undefined,
    profile_name: `${remoteName.value.trim()}_sync`,
    schedule: schedule.value,
    remote_subpath: remoteSubpath.value.trim() || undefined,
  }

  try {
    await dedupeStore.createRemote(payload)
    await remotesStore.fetchRemotes(true)
    if (createProfile.value) {
      await profilesStore.fetchProfiles()
    }
    dedupeStore.closeWizard()
  } catch (e: any) {
    errorMsg.value = e.message || 'Remote deployment failed'
  } finally {
    submitting.value = false
  }
}
</script>
