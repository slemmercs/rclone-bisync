<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { useRemotesStore } from '../stores/remotes'

const remotesStore = useRemotesStore()
const showGuide = ref(false)

function handleClose() {
  remotesStore.closeMigrationWizard()
}

async function handleSave() {
  await remotesStore.saveMigrationCredentials()
}
</script>

<template>
  <Dialog
    :visible="remotesStore.migrationModal.show"
    @update:visible="(val) => remotesStore.migrationModal.show = val"
    modal
    :style="{ width: '38rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-4 max-h-[80vh] overflow-y-auto' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-400 shrink-0">
          <i class="pi pi-key text-base"></i>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-sm font-bold text-white">OAuth App Configuration</h3>
            <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-carbon-bg text-amber-300 border border-amber-500/30">
              {{ remotesStore.migrationModal.remote }}
            </span>
          </div>
          <div class="text-[11px] text-slate-400">Configure personal OAuth credentials to avoid service limits</div>
        </div>
      </div>
    </template>

    <div class="space-y-4">
      <!-- Guidance Alert -->
      <div class="p-3.5 rounded-xl bg-carbon-bg border border-carbon-border space-y-1.5 text-xs">
        <div class="font-semibold text-slate-200 flex items-center gap-1.5">
          <i class="pi pi-info-circle text-electric-blue text-xs"></i>
          <span>Why configure personal credentials?</span>
        </div>
        <p class="text-slate-400 leading-relaxed text-[11px]">
          Using your own personal OAuth client credentials provides dedicated API rate limits, isolates your remote from shared quotas, and prevents third-party throttling across heavy sync workloads.
        </p>
      </div>

      <!-- Collapsible Setup Guide -->
      <div class="border border-carbon-border rounded-xl p-3 bg-carbon-bg/60 space-y-2">
        <div
          class="flex items-center justify-between cursor-pointer text-xs font-semibold text-slate-300"
          @click="showGuide = !showGuide"
        >
          <span class="flex items-center gap-2 uppercase tracking-wider text-[10px] text-slate-400">
            <i class="pi pi-book text-xs text-electric-blue"></i>
            <span>Setup Instructions</span>
          </span>
          <i :class="showGuide ? 'pi pi-chevron-up' : 'pi pi-chevron-down'" class="text-[10px] text-slate-400"></i>
        </div>

        <div v-show="showGuide" class="pt-2 border-t border-carbon-border/70 text-[11px] text-slate-400 space-y-2">
          <div v-if="remotesStore.migrationModal.type === 'drive'" class="space-y-1">
            <div class="text-white font-medium">Google Drive Steps:</div>
            <ol class="list-decimal pl-4 space-y-1">
              <li>Open <a href="https://console.cloud.google.com/" target="_blank" class="text-electric-blue underline">Google Cloud Console</a> and create or select a project.</li>
              <li>Enable the <strong>Google Drive API</strong> in APIs & Services.</li>
              <li>Configure the <strong>OAuth Consent Screen</strong> (External or Internal).</li>
              <li>Under <strong>Credentials</strong>, click <em>Create Credentials &rarr; OAuth client ID</em>.</li>
              <li>Select <strong>Desktop app</strong> as the application type.</li>
              <li>Copy the generated <strong>Client ID</strong> and <strong>Client Secret</strong> below.</li>
            </ol>
          </div>
          <div v-else-if="remotesStore.migrationModal.type === 'onedrive'" class="space-y-1">
            <div class="text-white font-medium">Microsoft OneDrive Steps:</div>
            <ol class="list-decimal pl-4 space-y-1">
              <li>Open the <a href="https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade" target="_blank" class="text-electric-blue underline">Azure App Registrations</a> console.</li>
              <li>Click <strong>New registration</strong>, name it, and select Supported account types.</li>
              <li>Add a Redirect URI for <strong>Mobile and desktop applications</strong> (<code class="text-white">http://localhost:53682/</code>).</li>
              <li>Copy the <strong>Application (client) ID</strong> and create a client secret under <em>Certificates & secrets</em>.</li>
            </ol>
          </div>
          <div v-else class="space-y-1">
            <div class="text-white font-medium">General OAuth App:</div>
            <p>Generate a dedicated Desktop / Public OAuth 2.0 Client ID and Secret in your provider's developer console and paste the credentials below.</p>
          </div>
        </div>
      </div>
      <!-- Credential Form Inputs -->
      <div class="space-y-3 font-mono text-xs">
        <div>
          <label class="block text-[11px] font-semibold text-slate-200 mb-1">OAuth Client ID</label>
          <input
            type="text"
            v-model="remotesStore.migrationModal.clientId"
            placeholder="Paste Client ID (e.g. 123456789-xxx.apps.googleusercontent.com)"
            class="w-full bg-carbon-bg border border-carbon-border rounded-xl px-3 py-2 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-electric-blue font-mono transition-colors"
          />
        </div>

        <div>
          <div class="flex justify-between items-center mb-1">
            <label class="text-[11px] font-semibold text-slate-200">OAuth Client Secret</label>
            <button
              type="button"
              @click="remotesStore.migrationModal.showSecret = !remotesStore.migrationModal.showSecret"
              class="text-[10px] text-electric-blue hover:text-sky-300 font-sans cursor-pointer"
            >
              {{ remotesStore.migrationModal.showSecret ? 'Hide Typed Secret' : 'Show Typed Secret' }}
            </button>
          </div>
          <input
            :type="remotesStore.migrationModal.showSecret ? 'text' : 'password'"
            v-model="remotesStore.migrationModal.clientSecret"
            placeholder="Paste Client Secret (e.g. GOCSPX-...)"
            class="w-full bg-carbon-bg border border-carbon-border rounded-xl px-3 py-2 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-electric-blue font-mono transition-colors"
          />
          <span class="text-[10px] text-slate-500 mt-1 block font-sans">
            For security, existing secrets cannot be read back. Leave blank to keep existing, or paste a new secret to update.
          </span>
        </div>

        <!-- Error & Success Feedback Alerts -->
        <div
          v-if="remotesStore.migrationModal.error"
          class="p-3 rounded-xl bg-rose-950/60 border border-rose-800/60 text-rose-300 text-xs flex items-center gap-2 font-sans"
        >
          <i class="pi pi-exclamation-circle text-rose-400 shrink-0"></i>
          <span>{{ remotesStore.migrationModal.error }}</span>
        </div>

        <div
          v-if="remotesStore.migrationModal.success"
          class="p-3 rounded-xl bg-emerald-950/60 border border-emerald-800/60 text-emerald-300 text-xs flex items-center gap-2 font-sans"
        >
          <i class="pi pi-check text-emerald-400 shrink-0"></i>
          <span>{{ remotesStore.migrationModal.success }}</span>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="pt-3 border-t border-carbon-border flex items-center justify-between">
        <span class="text-[10px] text-slate-500 font-mono">Persists securely to rclone.conf</span>
        <div class="flex items-center gap-2">
          <button
            type="button"
            @click="handleClose"
            :disabled="remotesStore.migrationModal.loading"
            class="px-4 py-2 bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border rounded-xl text-xs font-semibold transition-colors disabled:opacity-50 cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            @click="handleSave"
            :disabled="remotesStore.migrationModal.loading || !remotesStore.migrationModal.clientId.trim()"
            class="px-4 py-2 bg-electric-blue hover:bg-sky-400 text-black rounded-xl text-xs font-bold font-mono transition-colors flex items-center gap-2 disabled:opacity-50 cursor-pointer shadow"
          >
            <i v-if="remotesStore.migrationModal.loading" class="pi pi-spin pi-spinner text-xs"></i>
            <span>{{ remotesStore.migrationModal.loading ? 'Saving...' : 'Save Credentials' }}</span>
          </button>
        </div>
      </div>
    </div>
  </Dialog>
</template>
