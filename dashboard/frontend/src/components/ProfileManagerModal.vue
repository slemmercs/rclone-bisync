<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useProfilesStore } from '../stores/profiles'
import { useRemotesStore } from '../stores/remotes'
import type { Profile, ProfileVersion } from '../types/models'
import ModalProfileDelete from './ModalProfileDelete.vue'
import ProfileListTab from './profiles/ProfileListTab.vue'
import ProfileFormTab from './profiles/ProfileFormTab.vue'
import ProfileHistoryTab from './profiles/ProfileHistoryTab.vue'
import ProfileCloneDialog from './profiles/ProfileCloneDialog.vue'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ (e: 'update:visible', val: boolean): void; (e: 'open-remote-wizard'): void }>()

const profilesStore = useProfilesStore()
const remotesStore = useRemotesStore()
const profiles = computed(() => profilesStore.profiles)

const activeTab = ref<'list' | 'create' | 'edit' | 'history'>('list')
const editingProfile = ref<Partial<Profile>>({})
const formError = ref<string | null>(null)

const historyLoading = ref(false)
const historyRecords = ref<ProfileVersion[]>([])
const showCloneDialog = ref(false)
const cloneSource = ref<Profile | null>(null)
const showDeleteModal = ref(false)
const profileToDelete = ref<Profile | null>(null)

const normalizedRemotes = computed(() => {
  return remotesStore.remotes.map((r) => ({
    name: (r.name || '').replace(/:+$/, '').trim(),
    type: r.type || 'remote',
    rawName: r.name,
    underlying_remote: r.underlying_remote,
    underlying_path: r.underlying_path,
    crypt_overlay: r.crypt_overlay,
    hasher_overlay: r.hasher_overlay,
  }))
})

watch(
  () => props.visible,
  (val) => {
    if (val && remotesStore.remotes.length === 0) {
      remotesStore.fetchRemotes()
    }
  },
  { immediate: true }
)

function openCreate() {
  const rnd = Math.random().toString(36).substring(2, 8)
  const defRem = normalizedRemotes.value[0]?.name || 'local'
  editingProfile.value = {
    id: `prof_${rnd}`,
    name: '',
    local_path: defRem !== 'local' ? `~/Sync/${defRem}` : '~/Sync',
    remote: defRem,
    remote_path: '',
    schedule: 'manual',
    review_mode: 'deletions',
    enabled: true,
    flags: { compare_mode: 'size,modtime', conflict_resolve: 'newer', custom_flags: '', provider_options: {} },
    safety: { max_delete_percent: 10, canary_protection: true },
    exclude_patterns: ['.git/**', '*.tmp', '.DS_Store'],
  }
  formError.value = null
  activeTab.value = 'create'
}
function openEdit(p: Profile) {
  editingProfile.value = JSON.parse(JSON.stringify(p))
  editingProfile.value.remote = (editingProfile.value.remote || '').replace(/:+$/, '').trim()
  if (!editingProfile.value.flags) editingProfile.value.flags = {}
  if (!editingProfile.value.flags.provider_options) editingProfile.value.flags.provider_options = {}

  if (!editingProfile.value.safety) {
    editingProfile.value.safety = { max_delete_percent: 10, canary_protection: true }
  } else if (
    editingProfile.value.safety.max_delete_percent === undefined &&
    (editingProfile.value.safety as any).delete_threshold_pct !== undefined
  ) {
    editingProfile.value.safety.max_delete_percent = (editingProfile.value.safety as any).delete_threshold_pct
  }

  formError.value = null
  activeTab.value = 'edit'
}

async function openHistory(p: Profile) {
  editingProfile.value = p
  activeTab.value = 'history'
  historyLoading.value = true
  try {
    historyRecords.value = await profilesStore.fetchProfileHistory(p.id)
  } catch (err: any) {
    formError.value = err.message || 'Failed'
  } finally {
    historyLoading.value = false
  }
}

async function rollbackToVersion(ver: number) {
  if (!editingProfile.value.id) return
  try {
    await profilesStore.rollbackProfile(editingProfile.value.id, ver)
    activeTab.value = 'list'
  } catch (err: any) {
    alert(err.message || 'Rollback failed')
  }
}

function openClone(p: Profile) {
  cloneSource.value = p
  showCloneDialog.value = true
}

function confirmDelete(p: Profile) {
  profileToDelete.value = p
  showDeleteModal.value = true
}

async function handleSaveProfile(p: Partial<Profile>, excludeInput: string) {
  formError.value = null
  if (!p.name?.trim()) {
    formError.value = 'Profile name required'
    return
  }
  if (!p.local_path?.trim()) {
    formError.value = 'Local path required'
    return
  }
  if (!p.remote?.trim()) {
    formError.value = 'Storage remote required'
    return
  }

  p.name = p.name.trim()
  p.remote = (p.remote || '').replace(/:+$/, '').trim()
  if (p.remote_path) {
    p.remote_path = p.remote_path.replace(/^:+/, '').trim()
  }
  p.exclude_patterns = excludeInput
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)

  if (p.safety) {
    const pct = Number(p.safety.max_delete_percent ?? 10)
    p.safety.max_delete_percent = pct
    ;(p.safety as any).delete_threshold_pct = pct
  }

  if (p.flags?.provider_options) {
    const cleanedOpts: Record<string, string> = {}
    for (const [k, v] of Object.entries(p.flags.provider_options)) {
      if (v !== undefined && v !== null && v !== '') {
        cleanedOpts[k] = String(v)
      }
    }
    p.flags.provider_options = cleanedOpts
  }

  try {
    if (activeTab.value === 'create') await profilesStore.createProfile(p as Profile)
    else if (p.id) await profilesStore.updateProfile(p.id, p as Profile)
    activeTab.value = 'list'
  } catch (e: any) {
    formError.value = e.message || 'Failed to save'
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
          class="px-3 py-1 rounded-lg text-xs font-semibold cursor-pointer transition"
          :class="activeTab === 'list' ? 'bg-electric-blue text-white' : 'text-slate-400 hover:text-white'"
        >
          <span>Active Profiles</span>
          <span class="ml-1.5 text-[10px] px-1.5 py-0.2 rounded-full bg-carbon-elevated text-slate-300 font-mono">
            {{ profiles.length }}
          </span>
        </button>
        <button
          v-if="activeTab === 'create' || activeTab === 'edit'"
          class="px-3 py-1 rounded-lg text-xs font-semibold bg-electric-blue text-white"
        >
          {{ activeTab === 'create' ? 'Create Profile' : 'Edit Configuration' }}
        </button>
        <button
          v-if="activeTab === 'history'"
          class="px-3 py-1 rounded-lg text-xs font-semibold bg-electric-blue text-white flex items-center gap-1.5"
        >
          <i class="pi pi-history text-xs"></i><span>Version History</span>
        </button>
      </div>

      <button
        v-if="activeTab === 'list'"
        @click="openCreate()"
        class="px-3 py-1 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 cursor-pointer shadow-sm transition"
      >
        <i class="pi pi-plus text-xs"></i><span>New Profile</span>
      </button>
    </div>

    <div v-if="formError" class="p-2.5 bg-rose-500/10 border border-rose-500/20 rounded-xl text-xs text-rose-300 font-mono">
      {{ formError }}
    </div>

    <!-- TAB 1: ACTIVE PROFILES LIST -->
    <ProfileListTab
      v-if="activeTab === 'list'"
      :profiles="profiles"
      @sync="profilesStore.triggerSync()"
      @history="openHistory"
      @clone="openClone"
      @edit="openEdit"
      @delete="confirmDelete"
      @create="openCreate"
    />

    <!-- TAB 2: VERSION HISTORY -->
    <ProfileHistoryTab
      v-else-if="activeTab === 'history'"
      :profile="editingProfile"
      :records="historyRecords"
      :loading="historyLoading"
      @back="activeTab = 'list'"
      @rollback="rollbackToVersion"
    />

    <!-- TAB 3: CREATE / EDIT PROFILE FORM -->
    <ProfileFormTab
      v-else
      :is-create="activeTab === 'create'"
      :profile="editingProfile"
      :normalized-remotes="normalizedRemotes"
      @save="handleSaveProfile"
      @cancel="activeTab = 'list'"
      @open-remote-wizard="emit('open-remote-wizard')"
    />

    <!-- CLONE PROFILE DIALOG -->
    <ProfileCloneDialog
      v-model:visible="showCloneDialog"
      :source="cloneSource"
      @cloned="activeTab = 'list'"
    />

    <!-- DELETE PROFILE SAFEGUARD MODAL -->
    <ModalProfileDelete
      v-model:visible="showDeleteModal"
      :profile="profileToDelete"
      @deleted="activeTab = 'list'"
    />
  </Dialog>
</template>
