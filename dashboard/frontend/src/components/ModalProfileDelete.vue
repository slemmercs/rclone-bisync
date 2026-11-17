<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { useProfilesStore } from '../stores/profiles'
import type { Profile } from '../types/models'

const props = defineProps<{
  visible: boolean
  profile: Profile | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'deleted', id: string): void
}>()

const profilesStore = useProfilesStore()
const deleting = ref(false)
const error = ref<string | null>(null)

async function executeDelete() {
  if (!props.profile) return
  deleting.value = true
  error.value = null
  try {
    await profilesStore.deleteProfile(props.profile.id)
    emit('deleted', props.profile.id)
    emit('update:visible', false)
  } catch (err: any) {
    error.value = err.message || 'Failed to delete profile'
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    :style="{ width: '32rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-rose-500/30 rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-4' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 shrink-0">
          <i class="pi pi-trash text-lg"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white">Delete Profile Workload</h3>
          <div class="text-[11px] text-slate-400">Irreversible configuration removal</div>
        </div>
      </div>
    </template>

    <div v-if="profile" class="bg-carbon-bg border border-carbon-border rounded-xl p-3 space-y-1.5 font-mono text-xs">
      <div class="flex items-center justify-between text-slate-400">
        <span class="text-slate-500">Profile Name:</span>
        <span class="text-white font-semibold truncate max-w-[14rem]">{{ profile.name }}</span>
      </div>
      <div class="flex items-center justify-between text-slate-400">
        <span class="text-slate-500">Profile ID:</span>
        <span class="text-indigo-400 font-mono">{{ profile.id }}</span>
      </div>
      <div class="flex items-center justify-between text-slate-400">
        <span class="text-slate-500">Remote Target:</span>
        <span class="text-emerald-400">{{ profile.remote }}{{ profile.remote_path ? ':' + profile.remote_path : '' }}</span>
      </div>
      <div class="flex items-center justify-between text-slate-400">
        <span class="text-slate-500">Local Directory:</span>
        <span class="text-slate-300 truncate max-w-[14rem]">{{ profile.local_path }}</span>
      </div>
    </div>

    <div class="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl space-y-1 text-xs">
      <p class="font-semibold text-rose-300 flex items-center gap-1.5">
        <i class="pi pi-exclamation-triangle text-rose-400"></i>
        <span>Permanent Removal Safeguard</span>
      </p>
      <p class="leading-relaxed text-[11px] text-slate-300">
        Are you sure you want to permanently delete profile <span class="font-bold text-white font-mono">{{ profile?.name }}</span>?
        Background scheduler tickers for this profile will be terminated immediately. Local files will remain untouched.
      </p>
    </div>

    <div v-if="error" class="p-2.5 bg-rose-900/40 border border-rose-700 rounded-lg text-xs text-rose-200">
      {{ error }}
    </div>

    <div class="flex items-center justify-end gap-2 pt-2 border-t border-carbon-border">
      <button
        type="button"
        @click="emit('update:visible', false)"
        class="px-3 py-1.5 bg-carbon-elevated hover:bg-carbon-border text-slate-300 rounded-lg text-xs font-medium transition cursor-pointer"
      >
        Cancel
      </button>
      <button
        type="button"
        @click="executeDelete()"
        :disabled="deleting"
        class="px-3 py-1.5 bg-rose-600 hover:bg-rose-500 text-white rounded-lg text-xs font-semibold transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
      >
        <i v-if="deleting" class="pi pi-spin pi-spinner text-xs"></i>
        <span>{{ deleting ? 'Deleting...' : 'Permanently Delete Profile' }}</span>
      </button>
    </div>
  </Dialog>
</template>
