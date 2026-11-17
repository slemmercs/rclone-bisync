<template>
  <Dialog
    :visible="settingsStore.showGlobalSettingsModal"
    @update:visible="val => !val && settingsStore.closeGlobalSettingsModal()"
    modal
    header="Global Fleet Settings"
    :style="{ width: '92vw', maxWidth: '720px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border shadow-2xl rounded-2xl overflow-hidden' },
      header: { class: 'p-4 border-b border-carbon-border bg-carbon-surface text-white flex items-center justify-between' },
      content: { class: 'p-5 bg-carbon-canvas text-xs font-mono space-y-4 max-h-[75vh] overflow-y-auto' },
      footer: { class: 'p-4 border-t border-carbon-border bg-carbon-surface flex justify-between items-center' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-xl bg-electric-blue/10 border border-electric-blue/30 text-electric-blue flex items-center justify-center shrink-0">
          <i class="pi pi-cog text-base"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white tracking-wide">Global Fleet Settings</h3>
          <p class="text-[11px] text-slate-400">Daemon engine tuning, fallback flags, and alert integrations</p>
        </div>
      </div>
    </template>

    <!-- Tab Navigation Pills -->
    <div class="flex flex-wrap items-center gap-1.5 border-b border-carbon-border pb-3">
      <button
        v-for="t in tabs"
        :key="t.id"
        type="button"
        @click="activeTab = t.id"
        :class="activeTab === t.id ? 'bg-electric-blue text-black font-bold shadow' : 'bg-carbon-surface text-slate-400 hover:text-white border border-carbon-border'"
        class="px-3 py-1.5 rounded-lg transition-colors cursor-pointer text-xs flex items-center gap-1.5"
      >
        <i :class="t.icon" class="text-xs"></i>
        <span>{{ t.label }}</span>
      </button>
    </div>

    <!-- Error Banner -->
    <div v-if="saveError || settingsStore.configError" class="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs">
      {{ saveError || settingsStore.configError }}
    </div>

    <!-- TAB 1: OPERATIONAL DEFAULTS -->
    <div v-if="activeTab === 'defaults'" class="space-y-4">
      <div class="text-[11px] text-slate-400">Baseline configuration applied when creating new sync profiles.</div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Default Max Delete %</label>
          <input
            type="number"
            v-model.number="formDefaults.max_delete"
            min="0"
            max="100"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <span class="text-[10px] text-slate-500 mt-1 block">0 = Abort on any deletion; 1-100 = allowable % threshold.</span>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Conflict Resolution Mode</label>
          <select
            v-model="formDefaults.conflict_resolve"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
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
            v-model="formDefaults.compare_mode"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
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
            :value="formDefaults.canary_validation ? 'true' : 'false'"
            @change="formDefaults.canary_validation = ($event.target as HTMLSelectElement).value === 'true'"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="true">Enabled (Strict RCLONE_TEST Guard)</option>
            <option value="false">Disabled (Bypass Probing)</option>
          </select>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Empty Directory Cleanup</label>
          <div class="flex items-center gap-4 pt-1">
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="formDefaults.remove_empty_dirs" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
              <span class="text-xs text-slate-300">Remove Empty Dirs</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="formDefaults.create_empty_dirs" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
              <span class="text-xs text-slate-300">Create Empty Dirs</span>
            </label>
          </div>
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Explorer Remote Scoping</label>
          <select
            :value="formDefaults.lock_explorer_remote ? 'true' : 'false'"
            @change="formDefaults.lock_explorer_remote = ($event.target as HTMLSelectElement).value === 'true'"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          >
            <option value="false">Unlocked (Freely select any remote)</option>
            <option value="true">Locked to Active Profile Remote</option>
          </select>
        </div>
      </div>
    </div>

    <!-- TAB 2: FLEET FLAGS & CONCURRENCY -->
    <div v-else-if="activeTab === 'flags'" class="space-y-4">
      <div class="text-[11px] text-slate-400">
        Baseline Rclone engine concurrency, rate limits, and network parameters applied to the background daemon.
        Process-global options (<span class="text-amber-300 font-mono">--tpslimit</span>, <span class="text-amber-300 font-mono">--checkers</span>) define the daemon baseline; profiles with custom overrides queue safely and dynamically switch rates.
        <div class="mt-1 text-slate-500 italic">Note: Profile-specific operational flags (e.g. rename tracking, checksums) and cloud provider options are configured within individual sync profiles.</div>
      </div>

      <FlagsBuilder v-model="formDefaults.custom_flags" scopeFilter="engine" remoteType="all" />
    </div>

    <!-- TAB 2: ENGINE & POLLING -->
    <div v-else-if="activeTab === 'engine'" class="space-y-4">
      <div class="text-[11px] text-slate-400">Core daemon timeout and execution scheduler parameters.</div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Job Execution Timeout (s)</label>
          <input
            type="number"
            v-model.number="formDaemon.job_timeout_seconds"
            min="60"
            max="86400"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Stale Lock Timeout (s)</label>
          <input
            type="number"
            v-model.number="formDaemon.lock_stale_timeout_seconds"
            min="60"
            max="86400"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Rate Limit Backoff (hours)</label>
          <input
            type="number"
            v-model.number="formDaemon.rate_limit_backoff_hours"
            min="0.1"
            step="0.5"
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Job Poll Sleep (Initial / Max s)</label>
          <div class="flex items-center gap-2">
            <input
              type="number"
              v-model.number="formDaemon.job_poll_initial_sleep_sec"
              min="0.5"
              step="0.5"
              class="w-1/2 bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
            />
            <input
              type="number"
              v-model.number="formDaemon.job_poll_max_sleep_sec"
              min="1"
              step="1"
              class="w-1/2 bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- TAB 3: ALERT WEBHOOKS -->
    <div v-else-if="activeTab === 'notifications'" class="space-y-4">
      <div class="text-[11px] text-slate-400">External telemetry alert endpoints and heartbeat watchdogs.</div>

      <div class="space-y-3">
        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Alert Webhook URL</label>
          <input
            type="url"
            v-model="formNotifications.webhook_url"
            placeholder="https://hooks.slack.com/services/... or https://discord.com/api/webhooks/..."
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-300 mb-1">Watchdog Heartbeat URL</label>
          <input
            type="url"
            v-model="formNotifications.watchdog_ping_url"
            placeholder="https://nosnch.in/... or https://hc-ping.com/..."
            class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
      </div>
    </div>

    <!-- TAB 4: CONFIG ENCRYPTION -->
    <div v-else-if="activeTab === 'encryption'" class="space-y-4">
      <div class="flex items-center justify-between p-3 rounded-xl border bg-carbon-surface" :class="settingsStore.encryption.is_encrypted ? 'border-emerald-500/40' : 'border-carbon-border'">
        <div class="flex items-center gap-2.5">
          <i class="pi text-base" :class="settingsStore.encryption.is_encrypted ? 'pi-lock text-emerald-400' : 'pi-lock-open text-amber-400'"></i>
          <div>
            <div class="text-xs font-bold text-white uppercase">{{ settingsStore.encryption.is_encrypted ? 'Config Encrypted' : 'Plaintext Configuration' }}</div>
            <div class="text-[10px] text-slate-400">{{ settingsStore.encryption.config_file || 'Rclone config file' }}</div>
          </div>
        </div>
        <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded font-mono" :class="settingsStore.encryption.is_encrypted ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-amber-950 text-amber-400 border border-amber-800'">
          {{ settingsStore.encryption.is_encrypted ? 'Protected' : 'Unencrypted' }}
        </span>
      </div>

      <div v-if="encError" class="p-2.5 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs">
        {{ encError }}
      </div>

      <!-- Set / Change Encryption Passphrase -->
      <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <h4 class="text-xs font-bold text-white">{{ settingsStore.encryption.is_encrypted ? 'Modify Configuration Passphrase' : 'Set Configuration Passphrase' }}</h4>
        <div v-if="settingsStore.encryption.is_encrypted">
          <label class="block text-xs text-slate-400 mb-1">Current Passphrase</label>
          <input type="password" v-model="oldPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
        <div>
          <label class="block text-xs text-slate-400 mb-1">New Passphrase</label>
          <input type="password" v-model="newPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
        <div>
          <label class="block text-xs text-slate-400 mb-1">Confirm New Passphrase</label>
          <input type="password" v-model="confirmPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
        <button
          type="button"
          @click="handleSetPassword"
          :disabled="!newPassword || settingsStore.loadingEncryption"
          class="px-3.5 py-1.5 rounded-lg bg-electric-blue hover:bg-sky-400 text-black font-bold text-xs cursor-pointer disabled:opacity-50"
        >
          {{ settingsStore.loadingEncryption ? 'Applying...' : (settingsStore.encryption.is_encrypted ? 'Update Passphrase' : 'Save Passphrase') }}
        </button>
      </div>

      <!-- Remove Passphrase (if encrypted) -->
      <div v-if="settingsStore.encryption.is_encrypted" class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <h4 class="text-xs font-bold text-rose-300">Remove Encryption</h4>
        <div>
          <label class="block text-xs text-slate-400 mb-1">Current Passphrase</label>
          <input type="password" v-model="oldPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
        </div>
        <button
          type="button"
          @click="handleRemovePassword"
          :disabled="!oldPassword || settingsStore.loadingEncryption"
          class="px-3.5 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-bold text-xs cursor-pointer disabled:opacity-50"
        >
          Remove Passphrase
        </button>
      </div>
    </div>

    <template #footer>
      <button
        type="button"
        @click="settingsStore.closeGlobalSettingsModal()"
        class="px-3.5 py-1.5 rounded-xl border border-carbon-border text-slate-300 hover:text-white cursor-pointer"
      >
        Cancel
      </button>

      <button
        type="button"
        @click="handleSave"
        :disabled="settingsStore.savingConfig"
        class="px-4 py-1.5 rounded-xl bg-electric-blue hover:bg-sky-400 text-black font-bold cursor-pointer flex items-center gap-1.5 disabled:opacity-50"
      >
        <i class="pi" :class="settingsStore.savingConfig ? 'pi-spin pi-spinner' : 'pi-check'"></i>
        <span>{{ settingsStore.savingConfig ? 'Saving...' : 'Save Settings' }}</span>
      </button>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useSettingsStore } from '../stores/settings'
import FlagsBuilder from './FlagsBuilder.vue'

const settingsStore = useSettingsStore()
const activeTab = ref<'defaults' | 'flags' | 'engine' | 'notifications' | 'encryption'>('defaults')
const saveError = ref<string | null>(null)

const formDefaults = ref({ ...settingsStore.globalDefaults })
const formDaemon = ref({ ...settingsStore.daemonSettings })
const formNotifications = ref({ ...settingsStore.notificationSettings })

const newPassword = ref('')
const confirmPassword = ref('')
const oldPassword = ref('')
const daemonPassword = ref('')
const encError = ref<string | null>(null)

watch(() => settingsStore.globalDefaults, val => { formDefaults.value = { ...val } }, { deep: true })
watch(() => settingsStore.daemonSettings, val => { formDaemon.value = { ...val } }, { deep: true })
watch(() => settingsStore.notificationSettings, val => { formNotifications.value = { ...val } }, { deep: true })
watch(() => settingsStore.globalSettingsTab, tab => { activeTab.value = tab as any })

const tabs: Array<{ id: 'defaults' | 'flags' | 'engine' | 'notifications' | 'encryption'; label: string; icon: string }> = [
  { id: 'defaults', label: 'Defaults', icon: 'pi pi-sliders-h' },
  { id: 'flags', label: 'Fleet Flags', icon: 'pi pi-server' },
  { id: 'engine', label: 'Engine & Polling', icon: 'pi pi-clock' },
  { id: 'notifications', label: 'Alert Webhooks', icon: 'pi pi-bell' },
  { id: 'encryption', label: 'Encryption', icon: 'pi pi-lock' },
]

async function handleSave() {
  saveError.value = null
  try {
    await settingsStore.saveConfig({
      global_defaults: formDefaults.value,
      daemon: formDaemon.value,
      notifications: formNotifications.value,
    })
    settingsStore.closeGlobalSettingsModal()
  } catch (err: any) {
    saveError.value = err.message || 'Failed to save configuration'
  }
}

async function handleSetPassword() {
  encError.value = null
  if (newPassword.value !== confirmPassword.value) {
    encError.value = 'Passwords do not match'
    return
  }
  try {
    await settingsStore.setPassword(newPassword.value, oldPassword.value)
    newPassword.value = ''
    confirmPassword.value = ''
    oldPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to set encryption password'
  }
}

async function handleRemovePassword() {
  encError.value = null
  try {
    await settingsStore.removePassword(oldPassword.value)
    oldPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to remove password'
  }
}

async function handleSaveDaemonPassphrase() {
  encError.value = null
  try {
    await settingsStore.saveDaemonPassphrase(daemonPassword.value)
    daemonPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to save daemon passphrase'
  }
}
</script>
