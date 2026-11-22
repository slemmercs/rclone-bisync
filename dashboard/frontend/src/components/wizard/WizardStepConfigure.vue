<template>
  <div class="space-y-4">
    <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="text-xs text-slate-400">Selected Provider:</span>
        <span class="text-xs font-bold text-electric-blue uppercase font-mono">{{ provider?.name }}</span>
        <span
          v-if="isVirtual"
          class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20 ml-1"
        >
          Virtual Remote
        </span>
      </div>
      <button
        type="button"
        @click="emit('change-provider')"
        class="text-xs text-slate-400 hover:text-white cursor-pointer font-mono"
      >
        Change Provider
      </button>
    </div>

    <!-- Bisync Incompatibility Notice (union / combine) -->
    <div
      v-if="isBisyncBad"
      class="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20 text-xs text-amber-300 flex items-start gap-2.5"
    >
      <i class="pi pi-exclamation-triangle text-amber-400 text-base shrink-0 mt-0.5"></i>
      <div class="space-y-1">
        <span class="font-bold block text-white">{{ provider?.name?.toUpperCase() }} Bisync Incompatibility</span>
        <p class="text-[11px] text-amber-300/90 leading-relaxed font-sans">
          {{ provider?.name }} backends cannot be targeted for bidirectional sync due to asymmetric write ambiguity. Linked sync profile provisioning is disabled.
        </p>
      </div>
    </div>

    <div class="space-y-3 font-mono text-xs">
      <div>
        <label class="block text-slate-300 font-semibold mb-1">Remote Name *</label>
        <input
          type="text"
          v-model="form.remoteName"
          placeholder="e.g. gdrive_backup"
          class="w-full bg-carbon-surface border border-carbon-border rounded-lg px-3 py-1.5 text-white focus:outline-none focus:border-electric-blue"
        />
      </div>

      <!-- Authentication Configuration Section (OAuth Providers) -->
      <div v-if="isOAuthProvider(provider)" class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <label class="text-slate-200 font-semibold text-xs flex items-center gap-1.5">
            <i class="pi pi-shield text-emerald-400 text-xs"></i>
            <span>Authentication Mode</span>
          </label>
          <div class="inline-flex rounded-lg bg-carbon-canvas p-1 border border-carbon-border text-[11px]">
            <button
              type="button"
              @click="form.authMode = 'browser'"
              :class="form.authMode === 'browser' ? 'bg-electric-blue text-black font-bold shadow' : 'text-slate-400 hover:text-white'"
              class="px-2.5 py-1 rounded-md transition-all cursor-pointer flex items-center gap-1.5"
              title="Automatic browser sign-in"
            >
              <i class="pi pi-globe text-[11px]" :class="form.authMode === 'browser' ? 'text-black' : 'text-emerald-400'"></i>
              <span>Browser OAuth</span>
            </button>
            <button
              type="button"
              @click="form.authMode = 'custom'"
              :class="form.authMode === 'custom' ? 'bg-electric-blue text-black font-bold shadow' : 'text-slate-400 hover:text-white'"
              class="px-2.5 py-1 rounded-md transition-all cursor-pointer flex items-center gap-1.5"
              title="Custom Client ID and Secret"
            >
              <i class="pi pi-key text-[11px]" :class="form.authMode === 'custom' ? 'text-black' : 'text-amber-400'"></i>
              <span>Custom Credentials</span>
            </button>
          </div>
        </div>

        <div v-if="form.authMode === 'browser'" class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-300 flex items-start gap-2.5">
          <i class="pi pi-check-circle text-emerald-400 text-base shrink-0 mt-0.5"></i>
          <div class="space-y-1">
            <span class="font-bold block text-white">Interactive Browser Authorization</span>
            <p class="text-[11px] text-emerald-300/90 leading-relaxed font-sans">
              Upon deployment, Bisync Hub will coordinate with Rclone's local web server to launch a browser authorization window to sign into your {{ provider?.name }} account.
            </p>
          </div>
        </div>

        <div v-else-if="form.authMode === 'custom'" class="space-y-3 pt-1">
          <p class="text-[11px] text-slate-400 font-sans leading-relaxed">
            Recommended for Google Drive &amp; OneDrive to use your own cloud project credentials and avoid shared rate limits.
          </p>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label class="block text-slate-300 mb-1 text-[11px]">OAuth Client ID</label>
              <input
                type="text"
                v-model="form.customClientId"
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
                v-model="form.customClientSecret"
                placeholder="e.g. GOCSPX-..."
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- Non-OAuth Provider Schema-Driven Form -->
      <div v-else>
        <DynamicProviderForm :provider="provider" v-model="form.providerParams" />
      </div>

      <!-- Pipeline Visualization Badge (shown when virtual overlays are active) -->
      <div
        v-if="!isVirtual && (form.encryptRemote || form.accelerateHasher)"
        class="p-2.5 rounded-xl bg-carbon-surface border border-carbon-border/60 flex items-center justify-between gap-3 text-xs"
      >
        <span class="text-slate-400 text-[11px] font-sans flex items-center gap-1.5">
          <i class="pi pi-sitemap text-electric-blue text-xs"></i>
          Pipeline Topology:
        </span>
        <div class="flex items-center gap-1.5 font-mono text-[11px] overflow-x-auto">
          <span class="px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
            {{ form.remoteName.trim() || 'base' }}
          </span>
          <i class="pi pi-arrow-right text-[9px] text-slate-500"></i>
          <template v-if="form.encryptRemote">
            <span class="px-2 py-0.5 rounded bg-cyan-950/60 text-cyan-400 border border-cyan-800">
              {{ form.remoteName.trim() || 'base' }}_crypt
            </span>
            <i v-if="form.accelerateHasher" class="pi pi-arrow-right text-[9px] text-slate-500"></i>
          </template>
          <span v-if="form.accelerateHasher" class="px-2 py-0.5 rounded bg-violet-950/60 text-violet-400 border border-violet-800">
            {{ form.remoteName.trim() || 'base' }}_hasher
          </span>
        </div>
      </div>

      <!-- Client-Side Encryption (Crypt) Overlay -->
      <div v-if="!isVirtual" class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <div class="flex items-start justify-between gap-3">
          <label class="flex items-start gap-2.5 cursor-pointer select-none">
            <input
              type="checkbox"
              v-model="form.encryptRemote"
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

        <div v-if="form.encryptRemote" class="pt-3 border-t border-carbon-border/60 space-y-3 font-mono text-xs">
          <div class="p-2.5 rounded-lg bg-cyan-500/10 border border-cyan-500/20 text-cyan-300 text-[11px] font-sans flex items-center gap-2">
            <i class="pi pi-info-circle text-cyan-400 shrink-0"></i>
            <span>
              An encrypted virtual remote <strong>{{ form.remoteName.trim() ? form.remoteName.trim() + '_crypt' : 'remote_crypt' }}</strong> will be provisioned wrapping <strong>{{ form.remoteName.trim() || 'base_remote' }}</strong>.
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
                v-model="form.cryptPassword"
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
                v-model="form.cryptPassword2"
                placeholder="Optional salt passphrase"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-electric-blue"
              />
            </div>

            <div class="sm:col-span-2">
              <label class="block text-slate-300 font-semibold mb-1 text-[11px]">Filename Encryption Strategy</label>
              <select
                v-model="form.cryptFilenameEncryption"
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

      <!-- Hasher Acceleration Overlay -->
      <div v-if="!isVirtual" class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <div class="flex items-start justify-between gap-3">
          <label class="flex items-start gap-2.5 cursor-pointer select-none">
            <input
              type="checkbox"
              v-model="form.accelerateHasher"
              class="rounded bg-carbon-canvas border-carbon-border text-electric-blue focus:ring-0 w-4 h-4 mt-0.5"
            />
            <div class="space-y-0.5">
              <div class="text-xs text-white font-semibold flex items-center gap-1.5">
                <i class="pi pi-bolt text-violet-400 text-xs"></i>
                <span>Wrap with Hasher Acceleration</span>
                <span class="text-[9px] uppercase px-1.5 py-0.5 rounded font-bold bg-violet-500/10 text-violet-400 border border-violet-500/20">Hasher</span>
              </div>
              <p class="text-[11px] text-slate-400 font-sans leading-relaxed">
                Caches cryptographic checksums locally to accelerate bisync comparisons and eliminate slow cloud hash recalculations.
              </p>
            </div>
          </label>
        </div>

        <div v-if="form.accelerateHasher" class="pt-3 border-t border-carbon-border/60 space-y-3 font-mono text-xs">
          <div class="p-2.5 rounded-lg bg-violet-500/10 border border-violet-500/20 text-violet-300 text-[11px] font-sans flex items-center gap-2">
            <i class="pi pi-info-circle text-violet-400 shrink-0"></i>
            <span>
              A hasher overlay <strong>{{ form.remoteName.trim() ? form.remoteName.trim() + '_hasher' : 'remote_hasher' }}</strong> will be provisioned wrapping
              <strong>{{ form.encryptRemote ? (form.remoteName.trim() ? form.remoteName.trim() + '_crypt' : 'remote_crypt') : (form.remoteName.trim() || 'base_remote') }}</strong>.
            </span>
          </div>

          <div>
            <label class="block text-slate-300 font-semibold mb-1.5 text-[11px]">Cached Hash Algorithms</label>
            <div class="flex flex-wrap gap-2.5">
              <label
                v-for="hashType in ['md5', 'sha1', 'quickxor']"
                :key="hashType"
                class="flex items-center gap-2 cursor-pointer bg-carbon-canvas border border-carbon-border px-3 py-1.5 rounded-lg hover:border-violet-500/50 transition-colors"
              >
                <input
                  type="checkbox"
                  :value="hashType"
                  v-model="form.hasherHashes"
                  class="rounded bg-carbon-surface border-carbon-border text-violet-500 focus:ring-0 w-3.5 h-3.5"
                />
                <span class="text-xs font-mono uppercase font-bold" :class="form.hasherHashes.includes(hashType) ? 'text-violet-300' : 'text-slate-400'">
                  {{ hashType }}
                </span>
              </label>
            </div>
            <p v-if="form.hasherHashes.length === 0" class="text-[10px] text-rose-400 mt-1 font-sans">
              At least one hash algorithm must be selected.
            </p>
          </div>
        </div>
      </div>

      <!-- Sync Profile Bridge Toggle -->
      <div class="p-3 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
        <label class="flex items-center gap-2 select-none" :class="isBisyncBad ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'">
          <input
            type="checkbox"
            v-model="form.createProfile"
            :disabled="isBisyncBad"
            class="rounded bg-carbon-canvas border-carbon-border text-electric-blue focus:ring-0 w-3.5 h-3.5"
          />
          <span class="text-xs text-white font-semibold">Automatically provision linked sync profile</span>
          <span v-if="isBisyncBad" class="text-[10px] text-amber-400 font-normal font-sans">(Disabled for {{ provider?.name }})</span>
        </label>

        <div v-if="form.createProfile && !isBisyncBad" class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 border-t border-carbon-border/60">
          <div>
            <label class="block text-slate-400 mb-1 text-[11px]">Local Directory Path *</label>
            <input
              type="text"
              v-model="form.localPath"
              placeholder="e.g. /home/user/drive_sync"
              class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200"
            />
          </div>
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="block text-slate-400 text-[11px]">
                Remote Subpath <span v-if="isBucket && (form.encryptRemote || form.accelerateHasher)" class="text-rose-400">*</span>
              </label>
              <span v-if="isBucket && (form.encryptRemote || form.accelerateHasher)" class="text-[10px] text-rose-400 font-sans">
                Bucket root required
              </span>
            </div>
            <input
              type="text"
              v-model="form.remoteSubpath"
              :placeholder="isBucket ? 'e.g. my-bucket/vault (required)' : 'e.g. backup/vault (optional)'"
              :class="isBucket && (form.encryptRemote || form.accelerateHasher) && !form.remoteSubpath.trim() ? 'border-rose-500 focus:border-rose-400' : 'border-carbon-border focus:border-electric-blue'"
              class="w-full bg-carbon-canvas border rounded-lg px-2.5 py-1.5 text-slate-200"
            />
            <p v-if="isBucket && (form.encryptRemote || form.accelerateHasher) && !form.remoteSubpath.trim()" class="text-[10px] text-rose-400 mt-1 font-sans">
              Virtual overlays on bucket providers require a bucket name in remote subpath.
            </p>
          </div>
          <div>
            <label class="block text-slate-400 mb-1 text-[11px]">Sync Schedule</label>
            <select
              v-model="form.schedule"
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
          <div>
            <label class="block text-slate-400 mb-1 text-[11px]">Resolved Sync Target</label>
            <div class="w-full bg-carbon-canvas/50 border border-carbon-border rounded-lg px-2.5 py-1.5 text-electric-blue font-bold font-mono">
              {{ resolvedSyncTarget }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import DynamicProviderForm from '../DynamicProviderForm.vue'
import type { ProviderCatalogItem, WizardConfigForm } from '../../types'
import {
  isOAuthProvider,
  isVirtualProvider,
  isBisyncIncompatible,
  isBucketBasedProvider,
} from '../../types'

const props = defineProps<{
  provider: ProviderCatalogItem | null
  form: WizardConfigForm
}>()

const emit = defineEmits<{
  (e: 'change-provider'): void
}>()

const showSecret = ref<boolean>(false)
const showCryptPassword = ref<boolean>(false)
const showCryptSalt = ref<boolean>(false)

const isVirtual = computed(() => isVirtualProvider(props.provider))
const isBisyncBad = computed(() => isBisyncIncompatible(props.provider))
const isBucket = computed(() => isBucketBasedProvider(props.provider))

watch(
  () => props.provider,
  (p) => {
    if (isBisyncIncompatible(p)) {
      props.form.createProfile = false
    }
    if (isVirtualProvider(p)) {
      props.form.encryptRemote = false
      props.form.accelerateHasher = false
    }
  },
  { immediate: true }
)

const resolvedSyncTarget = computed(() => {
  const base = props.form.remoteName.trim() || 'base_remote'
  if (props.form.accelerateHasher) {
    return `${base}_hasher`
  }
  if (props.form.encryptRemote) {
    return `${base}_crypt`
  }
  return base
})
</script>
