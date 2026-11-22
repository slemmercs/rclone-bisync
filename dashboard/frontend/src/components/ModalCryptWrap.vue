<template>
  <Dialog
    :visible="remotesStore.cryptModal.show"
    @update:visible="(val) => remotesStore.cryptModal.show = val"
    modal
    :style="{ width: '38rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div class="p-2 rounded-xl bg-cyan-500/10 border border-cyan-500/30 text-cyan-400 flex items-center justify-center">
          <i class="pi pi-shield text-base"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white flex items-center gap-2">
            <span>Encrypt Remote with Crypt Overlay</span>
            <span v-if="remotesStore.cryptModal.remote" class="text-[10px] font-mono px-2 py-0.5 rounded bg-cyan-950/60 text-cyan-300 border border-cyan-800/60 font-semibold">
              {{ remotesStore.cryptModal.remote }}
            </span>
          </h3>
          <p class="text-[11px] text-slate-400 font-sans">
            Deploy client-side AES encrypted virtual storage atop your cloud backend.
          </p>
        </div>
      </div>
    </template>

    <div class="space-y-4 font-mono text-xs">
      <div class="p-3 rounded-xl bg-cyan-500/10 border border-cyan-500/20 text-cyan-300 text-[11px] font-sans flex items-center gap-2">
        <i class="pi pi-lock text-cyan-400 shrink-0 text-sm"></i>
        <span>
          Files and filenames are encrypted client-side before transmission. Cloud storage providers only ever store encrypted cipher blobs.
        </span>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <!-- Base Remote -->
        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">Underlying Remote *</label>
          <select
            v-if="availableBaseRemotes.length > 0"
            v-model="baseRemote"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-cyan-400 cursor-pointer"
          >
            <option v-for="r in availableBaseRemotes" :key="r.name" :value="r.name">{{ r.name }} ({{ r.type }})</option>
          </select>
          <input
            v-else
            type="text"
            v-model="baseRemote"
            placeholder="e.g. mydrive"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-cyan-400"
          />
        </div>

        <!-- Crypt Remote Name -->
        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">New Crypt Remote Name *</label>
          <input
            type="text"
            v-model="cryptName"
            placeholder="e.g. mydrive_crypt"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-cyan-400"
          />
        </div>
        <!-- Primary Password -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="text-slate-300 font-semibold text-[11px]">Primary Encryption Password *</label>
            <button
              type="button"
              @click="showPassword = !showPassword"
              class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans"
            >
              {{ showPassword ? 'Hide' : 'Show' }}
            </button>
          </div>
          <input
            :type="showPassword ? 'text' : 'password'"
            v-model="password"
            placeholder="Passphrase to encrypt files"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-cyan-400"
          />
        </div>

        <!-- Secondary Salt Passphrase -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="text-slate-300 font-semibold text-[11px]">Secondary Salt Passphrase (Optional)</label>
            <button
              type="button"
              @click="showSalt = !showSalt"
              class="text-[10px] text-electric-blue hover:text-sky-300 cursor-pointer font-sans"
            >
              {{ showSalt ? 'Hide' : 'Show' }}
            </button>
          </div>
          <input
            :type="showSalt ? 'text' : 'password'"
            v-model="password2"
            placeholder="Optional salt passphrase"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-cyan-400"
          />
        </div>

        <!-- Filename Encryption Mode -->
        <div>
          <label class="block text-slate-300 mb-1 text-[11px] font-semibold">Filename Encryption</label>
          <select
            v-model="filenameEncryption"
            class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-cyan-400 cursor-pointer"
          >
            <option value="standard">standard — Full filename encryption (Recommended)</option>
            <option value="obfuscate">obfuscate — Filename obfuscation (retains length)</option>
            <option value="off">off — Plaintext filenames</option>
          </select>
        </div>

        <!-- Directory Name Encryption -->
        <div class="flex items-center pt-5">
          <label class="flex items-center gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              v-model="directoryNameEncryption"
              :disabled="filenameEncryption === 'off'"
              class="rounded bg-carbon-canvas border-carbon-border text-cyan-400 focus:ring-0 w-3.5 h-3.5 disabled:opacity-40"
            />
            <span class="text-slate-300 text-[11px] font-semibold" :class="{ 'opacity-40': filenameEncryption === 'off' }">
              Encrypt Directory Names
            </span>
          </label>
        </div>
      </div>

      <div v-if="errorMsg" class="p-3 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl text-xs">
        {{ errorMsg }}
      </div>

      <div v-if="successMsg" class="p-3 bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 rounded-xl text-xs">
        {{ successMsg }}
      </div>

      <div class="flex items-center justify-end gap-2 pt-2 border-t border-carbon-border/60">
        <button
          type="button"
          @click="remotesStore.closeCryptWrap()"
          class="px-3.5 py-1.5 rounded-lg bg-carbon-canvas hover:bg-slate-800 text-slate-300 text-xs font-mono border border-carbon-border cursor-pointer transition-colors"
        >
          Cancel
        </button>
        <button
          type="button"
          @click="handleWrap"
          :disabled="!baseRemote || !cryptName || !password.trim() || wrapping"
          class="px-4 py-1.5 bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold rounded-lg text-xs font-mono transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-40"
        >
          <i class="pi" :class="wrapping ? 'pi-spin pi-spinner' : 'pi-shield'"></i>
          <span>{{ wrapping ? 'Encrypting...' : 'Provision Crypt Overlay' }}</span>
        </button>
      </div>
    </div>
  </Dialog>
</template>
<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useRemotesStore } from '../stores/remotes'
import type { CryptWrapRequest } from '../types'

const remotesStore = useRemotesStore()

const baseRemote = ref<string>('')
const cryptName = ref<string>('')
const password = ref<string>('')
const password2 = ref<string>('')
const filenameEncryption = ref<'standard' | 'obfuscate' | 'off'>('standard')
const directoryNameEncryption = ref<boolean>(true)
const showPassword = ref<boolean>(false)
const showSalt = ref<boolean>(false)

const wrapping = ref<boolean>(false)
const errorMsg = ref<string | null>(null)
const successMsg = ref<string | null>(null)

const availableBaseRemotes = computed(() => {
  return (remotesStore.remotes || []).filter(r => (r.type || '').toLowerCase() !== 'crypt')
})

watch(
  () => remotesStore.cryptModal.show,
  (show) => {
    if (!show) return
    const target = remotesStore.cryptModal.remote || ''
    baseRemote.value = target || (availableBaseRemotes.value[0]?.name || '')
    cryptName.value = baseRemote.value ? `${baseRemote.value}_crypt` : ''
    password.value = ''
    password2.value = ''
    filenameEncryption.value = 'standard'
    directoryNameEncryption.value = true
    showPassword.value = false
    showSalt.value = false
    errorMsg.value = null
    successMsg.value = null
  }
)

watch(baseRemote, (newBase) => {
  if (newBase && (!cryptName.value || cryptName.value.endsWith('_crypt'))) {
    cryptName.value = `${newBase}_crypt`
  }
})

async function handleWrap() {
  if (!baseRemote.value || !cryptName.value || !password.value.trim()) return
  wrapping.value = true
  errorMsg.value = null
  successMsg.value = null

  const req: CryptWrapRequest = {
    name: cryptName.value.trim(),
    remote: baseRemote.value.trim(),
    password: password.value.trim(),
    password2: password2.value.trim() || undefined,
    filename_encryption: filenameEncryption.value,
    directory_name_encryption: filenameEncryption.value !== 'off' && directoryNameEncryption.value,
  }

  try {
    await remotesStore.wrapCrypt(req)
    successMsg.value = `Successfully created encrypted Crypt remote "${req.name}" wrapping "${req.remote}"!`
    setTimeout(() => {
      remotesStore.closeCryptWrap()
    }, 1500)
  } catch (err: any) {
    errorMsg.value = err.message || 'Failed to wrap remote in Crypt'
  } finally {
    wrapping.value = false
  }
}
</script>

