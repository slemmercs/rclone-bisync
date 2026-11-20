<template>
  <Dialog
    :visible="settingsStore.showGlobalSettingsModal"
    @update:visible="val => !val && settingsStore.closeGlobalSettingsModal()"
    modal
    header="Global Fleet Settings"
    :style="{ width: '94vw', maxWidth: '980px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border shadow-2xl rounded-2xl overflow-hidden' },
      header: { class: 'p-4 border-b border-carbon-border bg-carbon-surface text-white flex items-center justify-between' },
      content: { class: 'p-0 bg-carbon-canvas text-xs font-mono overflow-hidden' },
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

    <div class="flex flex-col md:flex-row min-h-[500px] max-h-[74vh]">
      <!-- Left Sidebar Navigation -->
      <aside class="w-full md:w-56 shrink-0 border-b md:border-b-0 md:border-r border-carbon-border bg-carbon-surface/50 p-3 space-y-1 overflow-y-auto">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          @click="activeTab = t.id"
          :class="activeTab === t.id
            ? 'bg-electric-blue text-black font-bold shadow-md'
            : 'text-slate-400 hover:text-white hover:bg-carbon-surface border border-transparent'"
          class="w-full px-3 py-2.5 rounded-xl transition-all cursor-pointer text-xs flex items-center gap-2.5 text-left"
        >
          <i :class="t.icon" class="text-sm shrink-0"></i>
          <span class="truncate">{{ t.label }}</span>
        </button>
      </aside>

      <!-- Right Content Canvas -->
      <main class="flex-1 p-5 md:p-6 overflow-y-auto space-y-5">
        <!-- Error Banner -->
        <div v-if="saveError || settingsStore.configError" class="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs">
          {{ saveError || settingsStore.configError }}
        </div>

        <!-- TAB 1: OPERATIONAL DEFAULTS -->
        <SettingsDefaultsTab
          v-if="activeTab === 'defaults'"
          v-model="formDefaults"
        />
        <!-- TAB 2: FLEET FLAGS -->
        <div v-else-if="activeTab === 'flags'" class="space-y-4">
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-2">
            <h4 class="text-xs font-bold text-white">Global Engine Concurrency & Networking Flags</h4>
            <p class="text-[11px] text-slate-400">
              Baseline Rclone engine concurrency, rate limits, and network parameters applied to the background daemon. Process-global options (<code class="text-amber-400">--tpslimit</code>, <code class="text-amber-400">--checkers</code>) define the daemon baseline; profiles with custom overrides queue safely and dynamically switch rates.
            </p>
            <p class="text-[11px] text-slate-500 italic">
              Note: Profile-specific operational flags (e.g. rename tracking, checksums) and cloud provider options are configured within individual sync profiles.
            </p>
          </div>

          <FlagsBuilder v-model="formDefaults.custom_flags" scopeFilter="all" remoteType="all" />
        </div>

        <!-- TAB 3: ENGINE & POLLING -->
        <SettingsEngineTab
          v-else-if="activeTab === 'engine'"
          v-model="formDaemon"
        />
        <!-- TAB 4: LOGGING & CLEANUP -->
        <SettingsAuditTab
          v-else-if="activeTab === 'logging'"
          v-model:formLogging="formLogging"
          v-model:formCleanup="formCleanup"
        />

        <!-- TAB 5: TELEMETRY & FEATURES -->
        <SettingsTelemetryTab
          v-else-if="activeTab === 'telemetry'"
          v-model:formTelemetry="formTelemetry"
          v-model:formFeatures="formFeatures"
        />
        <!-- TAB 6: NOTIFICATIONS & WEBHOOKS -->
        <SettingsAlertsTab
          v-else-if="activeTab === 'notifications'"
          v-model="formNotifications"
        />

        <!-- TAB 7: ENCRYPTION -->
        <SettingsEncryptionTab v-else-if="activeTab === 'encryption'" />

        <!-- TAB 8: BANDWIDTH & LIMITS -->
        <SettingsBandwidthTab v-else-if="activeTab === 'bandwidth'" />
      </main>
    </div>

    <template #footer>
      <button
        type="button"
        @click="settingsStore.closeGlobalSettingsModal()"
        class="px-3.5 py-1.5 rounded-xl border border-carbon-border text-slate-300 hover:text-white cursor-pointer transition-colors"
      >
        Cancel
      </button>

      <button
        type="button"
        @click="handleSave"
        :disabled="settingsStore.savingConfig"
        class="px-4 py-1.5 rounded-xl bg-electric-blue hover:bg-sky-400 text-black font-bold cursor-pointer flex items-center gap-1.5 disabled:opacity-50 transition-colors"
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
import SettingsDefaultsTab from './settings/SettingsDefaultsTab.vue'
import SettingsEngineTab from './settings/SettingsEngineTab.vue'
import SettingsAuditTab from './settings/SettingsAuditTab.vue'
import SettingsTelemetryTab from './settings/SettingsTelemetryTab.vue'
import SettingsAlertsTab from './settings/SettingsAlertsTab.vue'
import SettingsEncryptionTab from './settings/SettingsEncryptionTab.vue'
import SettingsBandwidthTab from './settings/SettingsBandwidthTab.vue'

const settingsStore = useSettingsStore()
const activeTab = ref<'defaults' | 'flags' | 'engine' | 'bandwidth' | 'logging' | 'telemetry' | 'notifications' | 'encryption'>('defaults')
const saveError = ref<string | null>(null)

const formDefaults = ref({ ...settingsStore.globalDefaults })
const formDaemon = ref({ ...settingsStore.daemonSettings })
const formLogging = ref({ ...settingsStore.loggingSettings })
const formCleanup = ref({ ...settingsStore.cleanupSettings })
const formTelemetry = ref({ ...settingsStore.telemetrySettings })
const formFeatures = ref({ ...settingsStore.featureSettings })
const formNotifications = ref({ ...settingsStore.notificationSettings })

function resetForm() {
  formDefaults.value = { ...settingsStore.globalDefaults }
  formDaemon.value = { ...settingsStore.daemonSettings }
  formLogging.value = { ...settingsStore.loggingSettings }
  formCleanup.value = { ...settingsStore.cleanupSettings }
  formTelemetry.value = { ...settingsStore.telemetrySettings }
  formFeatures.value = { ...settingsStore.featureSettings }
  formNotifications.value = { ...settingsStore.notificationSettings }
  saveError.value = null
}

// Reset form data and discard dirty edits whenever the modal opens
watch(
  () => settingsStore.showGlobalSettingsModal,
  isOpen => {
    if (isOpen) {
      resetForm()
    }
  }
)

watch(() => settingsStore.globalDefaults, val => { formDefaults.value = { ...val } }, { deep: true })
watch(() => settingsStore.daemonSettings, val => { formDaemon.value = { ...val } }, { deep: true })
watch(() => settingsStore.loggingSettings, val => { formLogging.value = { ...val } }, { deep: true })
watch(() => settingsStore.cleanupSettings, val => { formCleanup.value = { ...val } }, { deep: true })
watch(() => settingsStore.telemetrySettings, val => { formTelemetry.value = { ...val } }, { deep: true })
watch(() => settingsStore.featureSettings, val => { formFeatures.value = { ...val } }, { deep: true })
watch(() => settingsStore.notificationSettings, val => { formNotifications.value = { ...val } }, { deep: true })
watch(() => settingsStore.globalSettingsTab, tab => { activeTab.value = tab as any })

const tabs: Array<{ id: 'defaults' | 'flags' | 'engine' | 'bandwidth' | 'logging' | 'telemetry' | 'notifications' | 'encryption'; label: string; icon: string }> = [
  { id: 'defaults', label: 'Defaults', icon: 'pi pi-sliders-h' },
  { id: 'flags', label: 'Fleet Flags', icon: 'pi pi-server' },
  { id: 'engine', label: 'Engine & Polling', icon: 'pi pi-clock' },
  { id: 'bandwidth', label: 'Bandwidth & Limits', icon: 'pi pi-gauge' },
  { id: 'logging', label: 'Logging & Cleanup', icon: 'pi pi-file' },
  { id: 'telemetry', label: 'Telemetry & Features', icon: 'pi pi-chart-bar' },
  { id: 'notifications', label: 'Alert Webhooks', icon: 'pi pi-bell' },
  { id: 'encryption', label: 'Encryption', icon: 'pi pi-lock' },
]

async function handleSave() {
  saveError.value = null
  try {
    await settingsStore.saveConfig({
      global_defaults: formDefaults.value,
      daemon: formDaemon.value,
      logging: formLogging.value,
      cleanup: formCleanup.value,
      telemetry: formTelemetry.value,
      features: formFeatures.value,
      notifications: formNotifications.value,
    })
    settingsStore.closeGlobalSettingsModal()
  } catch (err: any) {
    saveError.value = err.message || 'Failed to save configuration'
  }
}
</script>
