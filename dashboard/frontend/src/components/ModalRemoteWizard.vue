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
                <span v-if="isOAuthProvider(p)" class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">OAuth</span>
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
        <!-- Authentication Configuration Section (OAuth Providers) -->
        <div v-if="isOAuthProvider(selectedProvider)" class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <label class="text-slate-200 font-semibold text-xs flex items-center gap-1.5">
              <i class="pi pi-shield text-emerald-400 text-xs"></i>
              <span>Authentication Mode</span>
            </label>
            <!-- Mode Segmented Selector -->
            <div class="inline-flex rounded-lg bg-carbon-canvas p-1 border border-carbon-border text-[11px]">
              <button
                type="button"
                @click="authMode = 'browser'"
                :class="authMode === 'browser' ? 'bg-electric-blue text-black font-bold shadow' : 'text-slate-400 hover:text-white'"
                class="px-2.5 py-1 rounded-md transition-all cursor-pointer flex items-center gap-1.5"
                title="Automatic browser sign-in"
              >
                <i class="pi pi-globe text-[11px]" :class="authMode === 'browser' ? 'text-black' : 'text-emerald-400'"></i>
                <span>Browser OAuth</span>
              </button>
              <button
                type="button"
                @click="authMode = 'custom'"
                :class="authMode === 'custom' ? 'bg-electric-blue text-black font-bold shadow' : 'text-slate-400 hover:text-white'"
                class="px-2.5 py-1 rounded-md transition-all cursor-pointer flex items-center gap-1.5"
                title="Custom Client ID and Secret"
              >
                <i class="pi pi-key text-[11px]" :class="authMode === 'custom' ? 'text-black' : 'text-amber-400'"></i>
                <span>Custom Credentials</span>
              </button>
            </div>
          </div>

          <!-- Mode 1: Interactive Browser OAuth -->
          <div v-if="authMode === 'browser'" class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-300 flex items-start gap-2.5">
            <i class="pi pi-check-circle text-emerald-400 text-base shrink-0 mt-0.5"></i>
            <div class="space-y-1">
              <span class="font-bold block text-white">Interactive Browser Authorization</span>
              <p class="text-[11px] text-emerald-300/90 leading-relaxed font-sans">
                Upon deployment, Bisync Hub will coordinate with Rclone's local web server to launch a browser authorization window to sign into your {{ selectedProvider?.name }} account.
              </p>
            </div>
          </div>

          <!-- Mode 2: Custom Client Credentials -->
          <div v-else-if="authMode === 'custom'" class="space-y-3 pt-1">
            <p class="text-[11px] text-slate-400 font-sans leading-relaxed">
              Recommended for Google Drive &amp; OneDrive to use your own cloud project credentials and avoid shared rate limits.
            </p>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-slate-300 mb-1 text-[11px]">OAuth Client ID</label>
                <input
                  type="text"
                  v-model="customClientId"
                  placeholder="e.g. 123456789.apps.googleusercontent.com"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue"
                />
              </div>
              <div>
                <div class="flex justify-between items-center mb-1">
                  <label class="text-slate-300 text-[11px]">OAuth Client Secret</label>
                  <button
                    type="button"
                    @click="showSecret = !showSecret"
                    class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans"
                  >
                    {{ showSecret ? 'Hide' : 'Show' }}
                  </button>
                </div>
                <input
                  :type="showSecret ? 'text' : 'password'"
                  v-model="customClientSecret"
                  placeholder="e.g. GOCSPX-..."
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue"
                />
              </div>
            </div>
          </div>
        </div>
        <!-- Non-OAuth Provider Schema-Driven Form -->
        <div v-else>
          <DynamicProviderForm :provider="selectedProvider" v-model="providerParams" />
        </div>

        <!-- Client-Side Encryption (Crypt) Overlay -->
        <div class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
          <div class="flex items-start justify-between gap-3">
            <label class="flex items-start gap-2.5 cursor-pointer select-none">
              <input
                type="checkbox"
                v-model="encryptRemote"
                class="rounded bg-carbon-canvas border-carbon-border text-electric-blue focus:ring-0 w-4 h-4 mt-0.5"
              />
              <div class="space-y-0.5">
                <div class="text-xs text-white font-semibold flex items-center gap-1.5">
                  <i class="pi pi-shield text-cyan-400 text-xs"></i>
                  <span>Wrap with Client-Side Encryption (Crypt)</span>
                  <span class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">Crypt</span>
                </div>
                <p class="text-[11px] text-slate-400 font-sans leading-relaxed">
                  Transparently encrypt file contents and metadata on client side before upload. Zero plaintext reaches cloud storage.
                </p>
              </div>
            </label>
          </div>

          <!-- Crypt configuration options when enabled -->
          <div v-if="encryptRemote" class="pt-3 border-t border-carbon-border/60 space-y-3 font-mono text-xs">
            <div class="p-2.5 rounded-lg bg-cyan-500/10 border border-cyan-500/20 text-cyan-300 text-[11px] font-sans flex items-center gap-2">
              <i class="pi pi-info-circle text-cyan-400 shrink-0"></i>
              <span>
                An encrypted virtual remote <strong>{{ remoteName.trim() ? remoteName.trim() + '_crypt' : 'remote_crypt' }}</strong> will be provisioned wrapping <strong>{{ remoteName.trim() || 'base_remote' }}</strong>.
              </span>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <div class="flex items-center justify-between mb-1">
                  <label class="text-slate-300 font-semibold text-[11px]">Primary Encryption Password *</label>
                  <button
                    type="button"
                    @click="showCryptPassword = !showCryptPassword"
                    class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans"
                  >
                    {{ showCryptPassword ? 'Hide' : 'Show' }}
                  </button>
                </div>
                <input
                  :type="showCryptPassword ? 'text' : 'password'"
                  v-model="cryptPassword"
                  placeholder="Passphrase to encrypt files"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue"
                />
              </div>

              <div>
                <div class="flex items-center justify-between mb-1">
                  <label class="text-slate-300 font-semibold text-[11px]">Secondary Salt Passphrase (Optional)</label>
                  <button
                    type="button"
                    @click="showCryptSalt = !showCryptSalt"
                    class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans"
                  >
                    {{ showCryptSalt ? 'Hide' : 'Show' }}
                  </button>
                </div>
                <input
                  :type="showCryptSalt ? 'text' : 'password'"
                  v-model="cryptPassword2"
                  placeholder="Optional salt passphrase"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue"
                />
              </div>

              <div class="sm:col-span-2">
                <label class="block text-slate-300 font-semibold mb-1 text-[11px]">Filename Encryption Strategy</label>
                <select
                  v-model="cryptFilenameEncryption"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue cursor-pointer"
                >
                  <option value="standard">standard — Encrypt file and directory names (Recommended)</option>
                  <option value="obfuscate">obfuscate — Simple filename obfuscation (retains file length)</option>
                  <option value="off">off — Keep filenames in plaintext (encrypt data content only)</option>
                </select>
              </div>
            </div>
          </div>
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
          <div>
            <span class="text-slate-500">Authentication:</span>
            <span :class="isOAuthProvider(selectedProvider) ? 'text-emerald-400 font-semibold' : 'text-slate-300'">
              {{
                isOAuthProvider(selectedProvider)
                  ? (authMode === 'custom' ? 'Custom OAuth Credentials' : 'Browser OAuth (Interactive)')
                  : 'Standard / Direct'
              }}
            </span>
          </div>
          <div v-if="isOAuthProvider(selectedProvider) && authMode === 'custom' && customClientId">
            <span class="text-slate-500">Client ID:</span> {{ customClientId.slice(0, 16) }}...
          </div>
          <div>
            <span class="text-slate-500">Encryption:</span>
            <span :class="encryptRemote ? 'text-cyan-400 font-semibold' : 'text-slate-400'">
              {{ encryptRemote ? `Enabled (${remoteName.trim()}_crypt)` : 'Disabled' }}
            </span>
          </div>
          <div v-if="encryptRemote">
            <span class="text-slate-500">Filename Mode:</span>
            <span class="text-slate-300">{{ cryptFilenameEncryption }}</span>
          </div>
          <div><span class="text-slate-500">Linked Profile:</span> {{ createProfile ? 'Enabled' : 'Disabled' }}</div>
          <div v-if="createProfile">
            <span class="text-slate-500">Target Remote:</span>
            <span class="text-electric-blue">{{ encryptRemote ? `${remoteName.trim()}_crypt` : remoteName.trim() }}</span>
          </div>
          <div v-if="createProfile"><span class="text-slate-500">Local Path:</span> {{ localPath || '--' }}</div>
          <div v-if="createProfile"><span class="text-slate-500">Schedule:</span> {{ schedule }}</div>
        </div>
      </div>

      <div v-if="errorMsg" class="p-3 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl">
        {{ errorMsg }}
      </div>
    </div>

    <!-- Footer Controls -->
    <div class="flex items-center justify-between pt-3 pb-3 px-5 border-t border-carbon-border shrink-0 sticky -bottom-5 -mx-5 -mb-5 bg-carbon-surface/95 backdrop-blur-md z-10 shadow-lg">
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
import DynamicProviderForm from './DynamicProviderForm.vue'
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
const authMode = ref<'browser' | 'custom'>('browser')
const customClientId = ref<string>('')
const customClientSecret = ref<string>('')
const showSecret = ref<boolean>(false)
const providerParams = ref<Record<string, any>>({})
const createProfile = ref<boolean>(true)
const localPath = ref<string>('')
const remoteSubpath = ref<string>('')
const schedule = ref<string>('manual')

// Crypt Encryption Overlay State
const encryptRemote = ref<boolean>(false)
const cryptPassword = ref<string>('')
const cryptPassword2 = ref<string>('')
const cryptFilenameEncryption = ref<'standard' | 'obfuscate' | 'off'>('standard')
const showCryptPassword = ref<boolean>(false)
const showCryptSalt = ref<boolean>(false)

const errorMsg = ref<string | null>(null)
const submitting = ref<boolean>(false)

function isOAuthProvider(p: ProviderCatalogItem | null): boolean {
  if (!p) return false
  if (p.supports_oauth || (p as any).oauth_supported) return true
  const oauthTypes = ['drive', 'onedrive', 'dropbox', 'box', 'pcloud', 'hubic', 'yandex', 'mailru', 'google photos', 'zoho']
  const name = (p.name || '').toLowerCase()
  const prefix = (p.prefix || '').toLowerCase()
  return oauthTypes.includes(name) || oauthTypes.includes(prefix)
}

const canProceedToStep3 = computed(() => {
  if (!selectedProvider.value || !remoteName.value.trim()) return false
  if (encryptRemote.value && !cryptPassword.value.trim()) return false
  return true
})

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
  authMode.value = 'browser'
  customClientId.value = ''
  customClientSecret.value = ''
  showSecret.value = false
  providerParams.value = {}
  encryptRemote.value = false
  cryptPassword.value = ''
  cryptPassword2.value = ''
  cryptFilenameEncryption.value = 'standard'
  showCryptPassword.value = false
  showCryptSalt.value = false
  step.value = 2
}

async function handleDeploy() {
  if (!selectedProvider.value || !remoteName.value.trim()) return
  submitting.value = true
  errorMsg.value = null

  const cleanName = remoteName.value.trim()
  const params: Record<string, any> = { ...parameters.value }

  if (isOAuthProvider(selectedProvider.value)) {
    if (authMode.value === 'custom') {
      if (customClientId.value.trim()) params.client_id = customClientId.value.trim()
      if (customClientSecret.value.trim()) params.client_secret = customClientSecret.value.trim()
    }
  } else {
    Object.entries(providerParams.value).forEach(([k, v]) => {
      if (v !== undefined && v !== '') params[k] = v
    })
  }

  const payload: CreateRemotePayload = {
    name: cleanName,
    type: selectedProvider.value.name,
    parameters: params,
    create_profile: createProfile.value,
    local_path: localPath.value.trim() || undefined,
    profile_name: `${cleanName}_sync`,
    schedule: schedule.value,
    remote_subpath: remoteSubpath.value.trim() || undefined,
    encrypt_remote: encryptRemote.value,
    encrypted_overlay: encryptRemote.value,
    crypt_password: encryptRemote.value ? cryptPassword.value.trim() : undefined,
    crypt_password2: encryptRemote.value && cryptPassword2.value.trim() ? cryptPassword2.value.trim() : undefined,
    crypt_filename_encryption: encryptRemote.value ? cryptFilenameEncryption.value : undefined,
  }

  try {
    await dedupeStore.createRemote(payload)
    await remotesStore.fetchRemotes(true)
    if (createProfile.value) {
      await profilesStore.fetchProfiles()
    }
    dedupeStore.closeWizard()

    // If OAuth browser authentication was requested or required, launch interactive OAuth reconnect modal
    if (isOAuthProvider(selectedProvider.value)) {
      await remotesStore.reconnectRemote(cleanName)
    }
  } catch (e: any) {
    errorMsg.value = e.message || 'Remote deployment failed'
  } finally {
    submitting.value = false
  }
}
</script>
