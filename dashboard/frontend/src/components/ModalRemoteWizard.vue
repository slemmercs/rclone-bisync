<template>
  <Dialog
    :visible="remotesStore.wizardOpen"
    @update:visible="val => !val && remotesStore.closeWizard()"
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

    <!-- Stepper Panes -->
    <WizardStepProviderSelect
      v-if="step === 1"
      :providers="remotesStore.providers"
      :loading="remotesStore.providersLoading"
      :selected="selectedProvider"
      @select="selectProvider"
    />

    <WizardStepConfigure
      v-else-if="step === 2"
      :provider="selectedProvider"
      :form="form"
      @change-provider="step = 1"
    />

    <WizardStepReview
      v-else-if="step === 3"
      :provider="selectedProvider"
      :form="form"
      :error-msg="errorMsg"
    />

    <!-- Footer Controls -->
    <div class="flex items-center justify-between pt-3 pb-3 px-5 border-t border-carbon-border shrink-0 sticky -bottom-5 -mx-5 -mb-5 bg-carbon-surface/95 backdrop-blur-md z-10 shadow-lg">
      <button
        type="button"
        @click="step > 1 ? step-- : remotesStore.closeWizard()"
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
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useRemotesStore } from '../stores/remotes'
import { useProfilesStore } from '../stores/profiles'
import type { ProviderCatalogItem, CreateRemotePayload, WizardConfigForm } from '../types'
import { isOAuthProvider } from '../types'
import WizardStepProviderSelect from './wizard/WizardStepProviderSelect.vue'
import WizardStepConfigure from './wizard/WizardStepConfigure.vue'
import WizardStepReview from './wizard/WizardStepReview.vue'

const remotesStore = useRemotesStore()
const profilesStore = useProfilesStore()

const step = ref<number>(1)
const selectedProvider = ref<ProviderCatalogItem | null>(null)
const errorMsg = ref<string | null>(null)
const submitting = ref<boolean>(false)

function defaultForm(): WizardConfigForm {
  return {
    remoteName: '',
    authMode: 'browser',
    customClientId: '',
    customClientSecret: '',
    providerParams: {},
    encryptRemote: false,
    cryptPassword: '',
    cryptPassword2: '',
    cryptFilenameEncryption: 'standard',
    createProfile: true,
    localPath: '',
    remoteSubpath: '',
    schedule: 'manual',
  }
}

const form = ref<WizardConfigForm>(defaultForm())

watch(
  () => remotesStore.wizardOpen,
  (isOpen) => {
    if (isOpen) {
      if (remotesStore.selectedProvider) {
        selectProvider(remotesStore.selectedProvider)
      }
    } else {
      step.value = 1
      selectedProvider.value = null
      errorMsg.value = null
      form.value = defaultForm()
    }
  }
)

watch(
  () => remotesStore.selectedProvider,
  (p) => {
    if (p && remotesStore.wizardOpen) {
      selectProvider(p)
    }
  }
)

const canProceedToStep3 = computed(() => {
  if (!selectedProvider.value || !form.value.remoteName.trim()) return false
  if (form.value.encryptRemote && !form.value.cryptPassword.trim()) return false
  return true
})

function selectProvider(p: ProviderCatalogItem) {
  selectedProvider.value = p
  if (!form.value.remoteName) {
    form.value.remoteName = `${p.name}_remote`
  }
  form.value.authMode = 'browser'
  form.value.customClientId = ''
  form.value.customClientSecret = ''
  form.value.providerParams = {}
  form.value.encryptRemote = false
  form.value.cryptPassword = ''
  form.value.cryptPassword2 = ''
  form.value.cryptFilenameEncryption = 'standard'
  step.value = 2
}

async function handleDeploy() {
  if (!selectedProvider.value || !form.value.remoteName.trim()) return
  submitting.value = true
  errorMsg.value = null

  const cleanName = form.value.remoteName.trim()
  const params: Record<string, any> = {}

  if (isOAuthProvider(selectedProvider.value)) {
    if (form.value.authMode === 'custom') {
      if (form.value.customClientId.trim()) params.client_id = form.value.customClientId.trim()
      if (form.value.customClientSecret.trim()) params.client_secret = form.value.customClientSecret.trim()
    }
  } else {
    Object.entries(form.value.providerParams).forEach(([k, v]) => {
      if (v !== undefined && v !== '') params[k] = v
    })
  }

  const payload: CreateRemotePayload = {
    name: cleanName,
    type: selectedProvider.value.name,
    parameters: params,
    create_profile: form.value.createProfile,
    local_path: form.value.localPath.trim() || undefined,
    profile_name: `${cleanName}_sync`,
    schedule: form.value.schedule,
    remote_subpath: form.value.remoteSubpath.trim() || undefined,
    encrypt_remote: form.value.encryptRemote,
    encrypted_overlay: form.value.encryptRemote,
    crypt_password: form.value.encryptRemote ? form.value.cryptPassword.trim() : undefined,
    crypt_password2: form.value.encryptRemote && form.value.cryptPassword2.trim() ? form.value.cryptPassword2.trim() : undefined,
    crypt_filename_encryption: form.value.encryptRemote ? form.value.cryptFilenameEncryption : undefined,
  }

  try {
    await remotesStore.createRemote(payload)
    await remotesStore.fetchRemotes(true)
    if (form.value.createProfile) {
      await profilesStore.fetchProfiles()
    }
    remotesStore.closeWizard()

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
