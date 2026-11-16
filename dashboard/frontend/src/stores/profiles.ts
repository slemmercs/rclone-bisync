import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
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
      const res = await fetch('/api/profiles')
      if (!res.ok) {
        throw new Error(`Failed to fetch profiles: HTTP ${res.status}`)
      }
      const data = await res.json()
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
      const res = await fetch('/api/sync', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          profile_id: profileId,
          force: !!opts.force,
          dry_run: !!opts.dryRun,
          resync: !!opts.resync,
        }),
      })
      const data = await res.json()
      if (!res.ok) {
        fsmStore.transitionState(profileId, 'ABORTED', data.error || 'sync failed', data.error_type)
        throw new Error(data.error || 'Failed to trigger sync')
      }
      if (data.status === 'RUNNING' || data.status === 'SUCCESS') {
        fsmStore.transitionState(profileId, 'RUNNING', 'sync job running')
      }
      return data
    } catch (e: any) {
      fsmStore.transitionState(profileId, 'ABORTED', e.message)
      throw e
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
  }
})
