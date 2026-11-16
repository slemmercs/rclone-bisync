import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type { Profile } from '../types/models'
import { useFSMStore } from './fsm'

export const useProfilesStore = defineStore('profiles', () => {
  const profiles = ref<Profile[]>([])
  const selectedProfileId = ref<string | null>(null)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  const activeProfile = computed<Profile | null>(() => {
    if (!selectedProfileId.value && profiles.value.length > 0) {
      return profiles.value[0]
    }
    return profiles.value.find((p) => p.id === selectedProfileId.value) || null
  })

  const enabledProfiles = computed<Profile[]>(() => profiles.value.filter((p) => p.enabled && !p.archived))
  const archivedProfiles = computed<Profile[]>(() => profiles.value.filter((p) => p.archived))

  async function fetchProfiles() {
    loading.value = true
    error.value = null
    try {
      const data = await api.get('/api/profiles')
      profiles.value = Array.isArray(data) ? data : data.profiles || []
      if (!selectedProfileId.value && profiles.value.length > 0) {
        selectedProfileId.value = profiles.value[0].id
        const fsmStore = useFSMStore()
        fsmStore.setActiveProfile(profiles.value[0].id)
      }
    } catch (e: any) {
      error.value = e.message || 'Unknown error fetching profiles'
    } finally {
      loading.value = false
    }
  }

  function selectProfile(id: string) {
    selectedProfileId.value = id
    const fsmStore = useFSMStore()
    fsmStore.setActiveProfile(id)
  }

  async function triggerSync(opts: { force?: boolean; dryRun?: boolean; resync?: boolean } = {}) {
    const profileId = selectedProfileId.value || 'default'
    const fsmStore = useFSMStore()
    fsmStore.transitionState(profileId, 'QUEUED', 'sync dispatch initiated')

    try {
      let action = 'start_sync'
      if (opts.dryRun) action = 'dry_run'
      else if (opts.resync) action = 'resync'
      else if (opts.force) action = 'force_sync'

      const data = await api.post('/api/action', {
        action,
        profile_id: profileId,
      })

      if (data.status === 'error') {
        fsmStore.transitionState(profileId, 'ABORTED', data.msg || data.error || 'sync failed', data.error_type)
        throw new Error(data.msg || data.error || 'Failed to trigger sync')
      }

      if (data.status === 'requires_resync') {
        fsmStore.transitionState(profileId, 'ABORTED', data.msg || 'Initial sync requires resync confirmation')
        return data
      }

      fsmStore.transitionState(profileId, 'RUNNING', 'sync job running')
      return data
    } catch (e: any) {
      fsmStore.transitionState(profileId, 'ABORTED', e.message)
      throw e
    }
  }

  async function createProfile(payload: Partial<Profile>) {
    loading.value = true
    try {
      const created = await api.post<Profile>('/api/profiles', payload)
      await fetchProfiles()
      if (created?.id) selectProfile(created.id)
      return created
    } finally {
      loading.value = false
    }
  }

  async function updateProfile(id: string, payload: Partial<Profile>) {
    loading.value = true
    try {
      const updated = await api.put<Profile>(`/api/profiles/${id}`, payload)
      await fetchProfiles()
      return updated
    } finally {
      loading.value = false
    }
  }

  async function deleteProfile(id: string) {
    loading.value = true
    try {
      await api.delete(`/api/profiles/${id}`)
      await fetchProfiles()
      if (selectedProfileId.value === id) {
        selectedProfileId.value = profiles.value[0]?.id || null
      }
    } finally {
      loading.value = false
    }
  }

  async function cloneProfile(id: string, targetId: string, name: string) {
    loading.value = true
    try {
      const cloned = await api.post<Profile>(`/api/profiles/${id}/clone`, {
        target_id: targetId,
        name,
      })
      await fetchProfiles()
      if (cloned?.id) selectProfile(cloned.id)
      return cloned
    } finally {
      loading.value = false
    }
  }

  async function rollbackProfile(id: string, version: number) {
    loading.value = true
    try {
      const rolledBack = await api.post<Profile>(`/api/profiles/${id}/rollback`, { version })
      await fetchProfiles()
      return rolledBack
    } finally {
      loading.value = false
    }
  }

  return {
    profiles,
    selectedProfileId,
    loading,
    error,
    activeProfile,
    enabledProfiles,
    archivedProfiles,
    fetchProfiles,
    selectProfile,
    triggerSync,
    createProfile,
    updateProfile,
    deleteProfile,
    cloneProfile,
    rollbackProfile,
  }
})
