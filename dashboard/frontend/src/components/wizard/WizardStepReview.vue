<template>
  <div class="space-y-4 font-mono text-xs">
    <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
      <h4 class="font-bold text-white text-sm">Deployment Review</h4>
      <div class="grid grid-cols-2 gap-2 text-slate-300">
        <div><span class="text-slate-500">Remote Name:</span> {{ form.remoteName }}</div>
        <div><span class="text-slate-500">Provider:</span> {{ provider?.name }}</div>
        <div>
          <span class="text-slate-500">Authentication:</span>
          <span :class="isOAuth ? 'text-emerald-400 font-semibold' : 'text-slate-300'">
            {{
              isOAuth
                ? (form.authMode === 'custom' ? 'Custom OAuth Credentials' : 'Browser OAuth (Interactive)')
                : 'Standard / Direct'
            }}
          </span>
        </div>
        <div v-if="isOAuth && form.authMode === 'custom' && form.customClientId">
          <span class="text-slate-500">Client ID:</span> {{ form.customClientId.slice(0, 16) }}...
        </div>
        <div v-if="form.encryptRemote || form.accelerateHasher" class="col-span-2">
          <span class="text-slate-500">Storage Pipeline:</span>
          <span class="text-electric-blue font-bold ml-1.5">{{ pipelineSummary }}</span>
        </div>
        <div>
          <span class="text-slate-500">Encryption:</span>
          <span :class="form.encryptRemote ? 'text-cyan-400 font-semibold' : 'text-slate-400'">
            {{ form.encryptRemote ? `Enabled (${form.remoteName.trim()}_crypt)` : 'Disabled' }}
          </span>
        </div>
        <div v-if="form.encryptRemote">
          <span class="text-slate-500">Filename Mode:</span>
          <span class="text-slate-300">{{ form.cryptFilenameEncryption }}</span>
        </div>
        <div>
          <span class="text-slate-500">Hasher Cache:</span>
          <span :class="form.accelerateHasher ? 'text-violet-400 font-semibold' : 'text-slate-400'">
            {{ form.accelerateHasher ? `Enabled (${form.remoteName.trim()}_hasher)` : 'Disabled' }}
          </span>
        </div>
        <div v-if="form.accelerateHasher">
          <span class="text-slate-500">Cached Hashes:</span>
          <span class="text-slate-300 uppercase">{{ form.hasherHashes.join(', ') || 'None' }}</span>
        </div>
        <div><span class="text-slate-500">Linked Profile:</span> {{ form.createProfile ? 'Enabled' : 'Disabled' }}</div>
        <div v-if="form.createProfile">
          <span class="text-slate-500">Target Remote:</span>
          <span class="text-electric-blue font-bold">{{ resolvedTargetRemote }}</span>
        </div>
        <div v-if="form.createProfile"><span class="text-slate-500">Local Path:</span> {{ form.localPath || '--' }}</div>
        <div v-if="form.createProfile"><span class="text-slate-500">Remote Subpath:</span> {{ form.remoteSubpath || '(root)' }}</div>
        <div v-if="form.createProfile"><span class="text-slate-500">Schedule:</span> {{ form.schedule }}</div>
      </div>
    </div>

    <div v-if="errorMsg" class="p-3 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-xl">
      {{ errorMsg }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ProviderCatalogItem, WizardConfigForm } from '../../types'
import { isOAuthProvider } from '../../types'

const props = defineProps<{
  provider: ProviderCatalogItem | null
  form: WizardConfigForm
  errorMsg?: string | null
}>()

const isOAuth = computed(() => isOAuthProvider(props.provider))

const resolvedTargetRemote = computed(() => {
  const base = props.form.remoteName.trim() || 'remote'
  if (props.form.accelerateHasher) {
    return `${base}_hasher`
  }
  if (props.form.encryptRemote) {
    return `${base}_crypt`
  }
  return base
})

const pipelineSummary = computed(() => {
  const base = props.form.remoteName.trim() || 'base'
  const stages = [base]
  if (props.form.encryptRemote) stages.push(`${base}_crypt`)
  if (props.form.accelerateHasher) stages.push(`${base}_hasher`)
  return stages.join(' ➔ ')
})
</script>
