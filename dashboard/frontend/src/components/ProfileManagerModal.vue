<script setup lang="ts">
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import { useProfilesStore } from '../stores/profiles'
import type { Profile } from '../types/models'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ (e: 'update:visible', val: boolean): void }>()

const profilesStore = useProfilesStore()
const profiles = computed(() => profilesStore.profiles)
const loading = computed(() => profilesStore.loading)

const activeTab = ref<'list' | 'create' | 'edit'>('list')
const editingProfile = ref<Partial<Profile>>({})
const formError = ref<string | null>(null)

function openCreate() {
  editingProfile.value = {
    name: '',
    local_path: '',
    remote: 'local',
    remote_path: '',
    schedule: 'manual',
    review_mode: 'deletions',
    enabled: true,
    flags: {
      compare_mode: 'size,modtime',
      conflict_resolve: 'newer',
    },
    safety: {
      max_delete_percent: 50,
      canary_protection: true,
    },
  }
  formError.value = null
  activeTab.value = 'create'
}

function openEdit(p: Profile) {
  editingProfile.value = { ...p }
  formError.value = null
  activeTab.value = 'edit'
}

async function saveProfile() {
  formError.value = null
  try {
    if (activeTab.value === 'create') {
      await profilesStore.createProfile(editingProfile.value)
    } else if (editingProfile.value.id) {
      await profilesStore.updateProfile(editingProfile.value.id, editingProfile.value)
    }
    activeTab.value = 'list'
  } catch (e: any) {
    formError.value = e.message || 'Failed to save profile'
  }
}

async function deleteProfile(id: string) {
  if (!confirm(`Delete profile "${id}"?`)) return
  try {
    await profilesStore.deleteProfile(id)
  } catch (e: any) {
    alert(e.message || 'Failed to delete profile')
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    header="Sync Profiles Manager"
    :style="{ width: '46rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between text-white font-bold' },
      content: { class: 'pt-4 space-y-4' },
    }"
  >
    <div class="flex items-center justify-between border-b border-carbon-border pb-2">
      <div class="flex items-center gap-2">
        <button
          @click="activeTab = 'list'"
          class="px-3 py-1 rounded-lg text-xs font-semibold cursor-pointer"
          :class="activeTab === 'list' ? 'bg-electric-blue text-white' : 'text-slate-400 hover:text-white'"
        >
          Profiles ({{ profiles.length }})
        </button>
        <button v-if="activeTab !== 'list'" class="px-3 py-1 rounded-lg text-xs font-semibold bg-electric-blue text-white">
          {{ activeTab === 'create' ? 'New Profile' : 'Edit Profile' }}
        </button>
      </div>

      <button
        v-if="activeTab === 'list'"
        @click="openCreate()"
        class="px-2.5 py-1 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1 cursor-pointer"
      >
        <i class="pi pi-plus text-xs"></i><span>New Profile</span>
      </button>
    </div>

    <div v-if="formError" class="p-2.5 bg-rose-500/10 border border-rose-500/20 rounded-xl text-xs text-rose-300">
      {{ formError }}
    </div>

    <div v-if="activeTab === 'list'" class="max-h-[55vh] overflow-y-auto space-y-2 pr-1">
      <div v-for="p in profiles" :key="p.id" class="p-3 rounded-xl border border-carbon-border bg-carbon-elevated/40 flex items-center justify-between gap-3">
        <div class="space-y-0.5">
          <div class="flex items-center gap-2">
            <span class="font-bold text-sm text-white">{{ p.name }}</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded font-mono bg-carbon-elevated text-slate-400">v{{ p.version || 1 }}</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded font-bold uppercase bg-emerald-500/10 text-emerald-400">{{ p.state?.last_run_status || 'idle' }}</span>
          </div>
          <div class="text-xs font-mono text-slate-400 flex items-center gap-1">
            <span>{{ p.local_path }}</span><span>&harr;</span><span class="text-emerald-400">{{ p.remote }}:{{ p.remote_path || '' }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button @click="openEdit(p)" class="px-2.5 py-1 bg-carbon-elevated hover:bg-slate-800 text-slate-200 rounded-lg text-xs font-mono border border-carbon-border cursor-pointer">Edit</button>
          <button @click="deleteProfile(p.id)" class="px-2.5 py-1 bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 rounded-lg text-xs font-mono border border-rose-500/30 cursor-pointer">Delete</button>
        </div>
      </div>
    </div>

    <div v-else class="space-y-3 max-h-[55vh] overflow-y-auto pr-1">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
        <div>
          <label class="block text-slate-400 mb-1 font-semibold">Display Name</label>
          <input v-model="editingProfile.name" type="text" class="w-full bg-carbon-elevated border border-carbon-border rounded-xl p-2 text-white" placeholder="My Profile" />
        </div>
        <div>
          <label class="block text-slate-400 mb-1 font-semibold">Remote Name</label>
          <input v-model="editingProfile.remote" type="text" class="w-full bg-carbon-elevated border border-carbon-border rounded-xl p-2 text-white font-mono" placeholder="drive or backup" />
        </div>
        <div>
          <label class="block text-slate-400 mb-1 font-semibold">Local Path</label>
          <input v-model="editingProfile.local_path" type="text" class="w-full bg-carbon-elevated border border-carbon-border rounded-xl p-2 text-white font-mono" placeholder="/path/to/data" />
        </div>
        <div>
          <label class="block text-slate-400 mb-1 font-semibold">Remote Path</label>
          <input v-model="editingProfile.remote_path" type="text" class="w-full bg-carbon-elevated border border-carbon-border rounded-xl p-2 text-white font-mono" placeholder="subfolder" />
        </div>
      </div>
      <div class="flex items-center justify-end gap-2 pt-3 border-t border-carbon-border">
        <button @click="activeTab = 'list'" class="px-3 py-1.5 bg-carbon-elevated text-slate-300 rounded-xl text-xs cursor-pointer">Cancel</button>
        <button @click="saveProfile()" :disabled="loading" class="px-4 py-1.5 bg-electric-blue text-white rounded-xl text-xs font-semibold cursor-pointer disabled:opacity-50">Save</button>
      </div>
    </div>
  </Dialog>
</template>
