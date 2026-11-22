<script setup lang="ts">
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useProfilesStore } from '../../stores/profiles'
import type { Profile } from '../../types/models'

const props = defineProps<{
  visible: boolean
  source: Profile | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'cloned'): void
}>()

const profilesStore = useProfilesStore()
const cloneTargetId = ref('')
const cloneTargetName = ref('')
const cloneError = ref<string | null>(null)
const cloning = ref(false)

watch(
  () => props.source,
  (p) => {
    if (p) {
      cloneTargetId.value = `prof_${Math.random().toString(36).substring(2, 8)}`
      cloneTargetName.value = `${p.name} (Copy)`
      cloneError.value = null
    }
  },
  { immediate: true }
)

async function executeClone() {
  if (!props.source || !cloneTargetId.value.trim() || !cloneTargetName.value.trim()) return
  cloning.value = true
  cloneError.value = null
  try {
    await profilesStore.cloneProfile(
      props.source.id,
      cloneTargetId.value.trim(),
      cloneTargetName.value.trim()
    )
    emit('update:visible', false)
    emit('cloned')
  } catch (err: any) {
    cloneError.value = err.message || 'Clone failed'
  } finally {
    cloning.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    header="Clone Synchronization Workload"
    :style="{ width: '28rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-4' },
      header: { class: 'border-b border-carbon-border pb-2 text-white font-bold' },
      content: { class: 'pt-3 space-y-3' },
    }"
  >
    <div
      v-if="cloneError"
      class="p-2.5 bg-rose-500/10 border border-rose-500/20 rounded-lg text-xs text-rose-300 font-mono"
    >
      {{ cloneError }}
    </div>
    <div>
      <label class="block text-slate-400 mb-1 font-semibold text-xs">Target Profile Name</label>
      <input
        v-model="cloneTargetName"
        type="text"
        class="w-full bg-carbon-bg border border-carbon-border rounded-lg p-2 text-xs text-white"
      />
    </div>
    <div>
      <label class="block text-slate-400 mb-1 font-semibold text-xs">Target Profile ID</label>
      <input
        v-model="cloneTargetId"
        type="text"
        class="w-full bg-carbon-bg border border-carbon-border rounded-lg p-2 text-xs text-white font-mono"
      />
    </div>
    <div class="flex items-center justify-end gap-2 pt-2 border-t border-carbon-border">
      <button
        @click="emit('update:visible', false)"
        class="px-3 py-1.5 bg-carbon-elevated text-slate-300 rounded-lg text-xs cursor-pointer hover:bg-carbon-border transition"
      >
        Cancel
      </button>
      <button
        @click="executeClone()"
        :disabled="cloning"
        class="px-3.5 py-1.5 bg-electric-blue hover:bg-electric-blue/90 disabled:opacity-50 text-white rounded-lg text-xs font-semibold cursor-pointer transition shadow-sm"
      >
        {{ cloning ? 'Cloning...' : 'Clone Profile' }}
      </button>
    </div>
  </Dialog>
</template>
