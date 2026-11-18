<template>
  <div class="space-y-5">
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-2">
      <div class="flex items-center justify-between">
        <h4 class="text-xs font-bold text-white">Configuration Encryption at Rest</h4>
        <button
          type="button"
          @click="settingsStore.fetchEncryption()"
          :disabled="settingsStore.loadingEncryption"
          class="px-2.5 py-1 bg-carbon-canvas hover:bg-carbon-surface text-slate-300 hover:text-white rounded-lg text-[10px] font-medium border border-carbon-border transition-colors cursor-pointer flex items-center gap-1.5"
        >
          <i class="pi pi-refresh text-[10px]" :class="{ 'animate-spin': settingsStore.loadingEncryption }"></i>
          <span>Refresh</span>
        </button>
      </div>
      <p class="text-[11px] text-slate-400">
        Protect sensitive tokens, passwords, and custom paths in <code class="text-slate-300">bisync.env</code> using AES-256-GCM encryption with Argon2id key derivation.
      </p>
    </div>

    <!-- Encryption Status Banner -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl flex items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div
          class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0"
          :class="settingsStore.encryption.is_encrypted ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-slate-800 text-slate-400'"
        >
          <i :class="settingsStore.encryption.is_encrypted ? 'pi pi-lock' : 'pi pi-lock-open'" class="text-sm"></i>
        </div>
        <div>
          <span class="text-xs font-bold text-white block">
            {{ settingsStore.encryption.is_encrypted ? 'Configuration Encrypted' : 'Configuration Stored in Plaintext' }}
          </span>
          <span class="text-[10px] text-slate-500">
            {{ settingsStore.encryption.is_encrypted ? (settingsStore.encryption.config_file || 'Sensitive keys protected via local passphrase.') : 'Tokens and secrets are stored in plaintext on disk.' }}
          </span>
        </div>
      </div>
      <span
        class="text-[10px] uppercase font-mono px-2 py-0.5 rounded border"
        :class="settingsStore.encryption.is_encrypted ? 'bg-emerald-950 text-emerald-300 border-emerald-800/60' : 'bg-slate-800 text-slate-400 border-slate-700'"
      >
        {{ settingsStore.encryption.is_encrypted ? 'SECURE' : 'PLAINTEXT' }}
      </span>
    </div>

    <!-- Error Banner -->
    <div v-if="encError || settingsStore.encryptionError" class="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs flex items-center justify-between">
      <span>{{ encError || settingsStore.encryptionError }}</span>
      <button type="button" @click="encError = null" class="text-rose-400 hover:text-rose-200 text-xs">
        <i class="pi pi-times"></i>
      </button>
    </div>
    <!-- Daemon Secrets Persistence Card (rc-secrets.env) -->
    <div v-if="settingsStore.encryption.is_encrypted" class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <div class="flex items-center justify-between gap-2">
        <div>
          <h4 class="text-xs font-bold text-white">Daemon Passphrase Persistence</h4>
          <p class="text-[11px] text-slate-400 mt-0.5">
            Stores <code class="text-slate-300 font-mono">RCLONE_CONFIG_PASS</code> in <code class="text-slate-300 font-mono">rc-secrets.env</code> so the background daemon can execute scheduled syncs without prompts.
          </p>
        </div>
        <span
          class="text-[10px] uppercase font-mono px-2 py-0.5 rounded border shrink-0"
          :class="settingsStore.encryption.has_daemon_passphrase
            ? 'bg-emerald-950 text-emerald-300 border-emerald-800/60'
            : 'bg-amber-950 text-amber-300 border-amber-800/60 animate-pulse'"
        >
          {{ settingsStore.encryption.has_daemon_passphrase ? 'Configured (rc-secrets.env)' : 'Missing / Unconfigured' }}
        </span>
      </div>

      <div class="pt-2 border-t border-carbon-border space-y-2">
        <div class="flex flex-col sm:flex-row gap-2.5 items-stretch sm:items-center">
          <input
            type="password"
            v-model="daemonPassword"
            placeholder="Enter rclone.conf passphrase to persist for daemon"
            class="flex-1 bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
          <button
            type="button"
            @click="handleSaveDaemonPassphrase"
            :disabled="!daemonPassword.trim() || settingsStore.loadingEncryption"
            class="px-3.5 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-400 disabled:opacity-50 text-slate-950 font-bold text-xs transition-colors cursor-pointer shrink-0 shadow-sm flex items-center justify-center gap-1.5"
          >
            <i v-if="settingsStore.loadingEncryption" class="pi pi-spin pi-spinner text-xs"></i>
            <span>{{ settingsStore.loadingEncryption ? 'Persisting...' : 'Verify & Persist' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Set / Update Master Passphrase -->
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <h4 class="text-xs font-bold text-white">
        {{ settingsStore.encryption.is_encrypted ? 'Change Master Passphrase' : 'Set Master Passphrase' }}
      </h4>
      <div v-if="settingsStore.encryption.is_encrypted">
        <label class="block text-xs text-slate-400 mb-1">Current Passphrase</label>
        <input
          type="password"
          v-model="changeOldPassword"
          placeholder="Enter current passphrase"
          class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
        />
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label class="block text-xs text-slate-400 mb-1">New Passphrase</label>
          <input
            type="password"
            v-model="newPassword"
            placeholder="Enter new passphrase"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
        <div>
          <label class="block text-xs text-slate-400 mb-1">Confirm New Passphrase</label>
          <input
            type="password"
            v-model="confirmPassword"
            placeholder="Confirm new passphrase"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
          />
        </div>
      </div>
      <button
        type="button"
        @click="handleSetPassword"
        :disabled="!newPassword || settingsStore.loadingEncryption"
        class="px-3.5 py-1.5 rounded-lg bg-electric-blue hover:bg-sky-400 text-black font-bold text-xs cursor-pointer disabled:opacity-50 transition-colors"
      >
        {{ settingsStore.loadingEncryption ? 'Applying...' : (settingsStore.encryption.is_encrypted ? 'Update Passphrase' : 'Save Passphrase') }}
      </button>
    </div>
    <!-- Remove Passphrase (if encrypted) -->
    <div v-if="settingsStore.encryption.is_encrypted" class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <h4 class="text-xs font-bold text-rose-300">Remove Encryption</h4>
      <p class="text-[11px] text-slate-400">Decrypts the configuration and stores secrets in plaintext format on disk.</p>
      <div>
        <label class="block text-xs text-slate-400 mb-1">Current Passphrase</label>
        <input
          type="password"
          v-model="removeOldPassword"
          placeholder="Enter current passphrase to decrypt"
          class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
        />
      </div>
      <button
        type="button"
        @click="handleRemovePassword"
        :disabled="!removeOldPassword || settingsStore.loadingEncryption"
        class="px-3.5 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-bold text-xs cursor-pointer disabled:opacity-50 transition-colors"
      >
        Remove Passphrase
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useSettingsStore } from '../../stores/settings'

const settingsStore = useSettingsStore()

const changeOldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const removeOldPassword = ref('')
const daemonPassword = ref('')
const encError = ref<string | null>(null)

async function handleSetPassword() {
  encError.value = null
  if (newPassword.value !== confirmPassword.value) {
    encError.value = 'Passwords do not match'
    return
  }
  try {
    await settingsStore.setPassword(newPassword.value, changeOldPassword.value)
    newPassword.value = ''
    confirmPassword.value = ''
    changeOldPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to set encryption password'
  }
}

async function handleRemovePassword() {
  encError.value = null
  try {
    await settingsStore.removePassword(removeOldPassword.value)
    removeOldPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to remove password'
  }
}

async function handleSaveDaemonPassphrase() {
  encError.value = null
  if (!daemonPassword.value.trim()) {
    encError.value = 'Passphrase cannot be empty'
    return
  }
  try {
    await settingsStore.saveDaemonPassphrase(daemonPassword.value)
    daemonPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to save daemon passphrase'
  }
}
</script>
