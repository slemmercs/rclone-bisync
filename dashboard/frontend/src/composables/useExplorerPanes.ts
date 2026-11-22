import { ref, computed, watch, onMounted } from 'vue'
import { useProfilesStore } from '../stores/profiles'
import { useExplorerStore } from '../stores/explorer'
import { useSettingsStore } from '../stores/settings'
import { useRemotesStore } from '../stores/remotes'
import type { BrowseItem } from '../types/models'

export function useExplorerPanes() {
  const profilesStore = useProfilesStore()
  const explorerStore = useExplorerStore()
  const settingsStore = useSettingsStore()
  const remotesStore = useRemotesStore()

  const activeProfile = computed(() => profilesStore.activeProfile)
  const isRemoteLocked = computed(() => !!settingsStore.globalDefaults.lock_explorer_remote)

  // Pane Swap & Navigation State
  const isSwapped = ref(false)
  const leftPath = ref('')
  const leftItems = ref<BrowseItem[]>([])
  const leftLoading = ref(false)
  const leftCanary = ref(false)
  const leftSelected = ref<BrowseItem | null>(null)
  const leftUploading = ref(false)

  const rightRemoteName = ref('')
  const rightPath = ref('')
  const rightItems = ref<BrowseItem[]>([])
  const rightLoading = ref(false)
  const rightCanary = ref(false)
  const rightSelected = ref<BrowseItem | null>(null)
  const rightUploading = ref(false)

  const leftIsLocal = computed(() => !isSwapped.value)
  const rightIsLocal = computed(() => isSwapped.value)
  const leftRemote = computed(() => (isSwapped.value ? rightRemoteName.value : 'local'))
  const rightRemote = computed(() => (isSwapped.value ? 'local' : rightRemoteName.value))

  onMounted(() => {
    remotesStore.fetchRemotes()
    settingsStore.fetchConfig()
  })

  async function applyTargetJump() {
    const target = explorerStore.consumeTargetJumpPath()
    if (!target) return

    let dir = target.dir ? target.dir.trim().replace(/^["'`\s]+|["'`\s]+$/g, '').replace(/\\/g, '/').replace(/^\/+|\/+$/g, '') : ''
    const profile = activeProfile.value

    if (profile?.local_path) {
      const cleanLocalBase = profile.local_path.trim().replace(/\\/g, '/').replace(/^\/+|\/+$/g, '')
      if (dir === cleanLocalBase) {
        dir = ''
      } else if (dir.startsWith(cleanLocalBase + '/')) {
        dir = dir.substring(cleanLocalBase.length + 1)
      }
    }

    const remoteSubpath = (profile?.remote_path || '').trim().replace(/\\/g, '/').replace(/^\/+|\/+$/g, '')
    let remoteDir = dir
    if (remoteSubpath) {
      if (!remoteDir) {
        remoteDir = remoteSubpath
      } else if (!remoteDir.startsWith(remoteSubpath)) {
        remoteDir = `${remoteSubpath}/${remoteDir}`
      }
    }

    if (isSwapped.value) {
      leftPath.value = remoteDir
      rightPath.value = dir
    } else {
      leftPath.value = dir
      rightPath.value = remoteDir
    }

    await refreshAll()
  }

  watch(
    () => activeProfile.value,
    (profile) => {
      if (profile) {
        rightRemoteName.value = (profile.remote || 'local').replace(/:$/, '')
        if (explorerStore.targetJumpPath) {
          applyTargetJump()
        } else {
          leftPath.value = ''
          rightPath.value = (profile.remote_path || '').replace(/^\/+/, '')
          refreshAll()
        }
        explorerStore.fetchUploadDockStatus(profile.id)
      }
    },
    { immediate: true }
  )

  watch(
    () => explorerStore.targetJumpPath,
    (target) => {
      if (target) {
        applyTargetJump()
      }
    }
  )

  function swapPanes() {
    isSwapped.value = !isSwapped.value
    const tempPath = leftPath.value
    leftPath.value = rightPath.value
    rightPath.value = tempPath

    const tempItems = leftItems.value
    leftItems.value = rightItems.value
    rightItems.value = tempItems

    const tempCanary = leftCanary.value
    leftCanary.value = rightCanary.value
    rightCanary.value = tempCanary
  }

  async function refreshPane(pane: 'left' | 'right') {
    const remote = pane === 'left' ? leftRemote.value : rightRemote.value
    const path = pane === 'left' ? leftPath.value : rightPath.value

    if (pane === 'left') {
      leftLoading.value = true
      leftSelected.value = null
    } else {
      rightLoading.value = true
      rightSelected.value = null
    }

    try {
      const data = await explorerStore.browse(remote, path, activeProfile.value?.id)
      const resItems = Array.isArray(data?.items) ? (data.items as BrowseItem[]) : []
      if (pane === 'left') {
        leftItems.value = resItems
        if (!path) leftCanary.value = !!data?.has_canary
      } else {
        rightItems.value = resItems
        if (!path) rightCanary.value = !!data?.has_canary
      }
    } catch {
      if (pane === 'left') leftItems.value = []
      else rightItems.value = []
    } finally {
      if (pane === 'left') leftLoading.value = false
      else rightLoading.value = false
    }
  }

  async function refreshAll() {
    await Promise.all([refreshPane('left'), refreshPane('right')])
  }

  function handleRemoteUpdate(newRemote: string) {
    rightRemoteName.value = newRemote.replace(/:$/, '')
    rightPath.value = ''
    const remotePane = isSwapped.value ? 'left' : 'right'
    refreshPane(remotePane)
  }

  function handleNavigate(pane: 'left' | 'right', path: string) {
    if (pane === 'left') {
      leftPath.value = path
      refreshPane('left')
    } else {
      rightPath.value = path
      refreshPane('right')
    }
  }

  function getRelativeItemPath(pane: 'left' | 'right', item: BrowseItem): string {
    const currentPath = pane === 'left' ? leftPath.value : rightPath.value
    return currentPath ? `${currentPath.replace(/\/+$/, '')}/${item.Name}` : item.Name
  }

  function upDirectory(pane: 'left' | 'right') {
    const current = pane === 'left' ? leftPath.value : rightPath.value
    const parts = current.replace(/\/+$/, '').split('/').filter(Boolean)
    if (parts.length > 0) {
      parts.pop()
      handleNavigate(pane, parts.join('/'))
    }
  }

  async function touchCanary(pane: 'left' | 'right') {
    const remote = pane === 'left' ? leftRemote.value : rightRemote.value
    try {
      await explorerStore.touchCanary(remote, '', activeProfile.value?.id)
      if (pane === 'left') leftCanary.value = true
      else rightCanary.value = true
      refreshPane(pane)
    } catch (e: any) {
      alert(e.message || 'Failed to touch canary')
    }
  }

  async function handleUploadFiles(pane: 'left' | 'right', files: File[]) {
    const remote = pane === 'left' ? leftRemote.value : rightRemote.value
    const path = pane === 'left' ? leftPath.value : rightPath.value

    if (pane === 'left') leftUploading.value = true
    else rightUploading.value = true

    try {
      for (const file of files) {
        await explorerStore.uploadFile(file, remote, path, activeProfile.value?.id)
      }
      refreshPane(pane)
      explorerStore.fetchUploadDockStatus(activeProfile.value?.id)
    } catch (err: any) {
      alert(err.message || 'File upload failed')
    } finally {
      if (pane === 'left') leftUploading.value = false
      else rightUploading.value = false
    }
  }

  return {
    activeProfile,
    isRemoteLocked,
    remotesStore,
    isSwapped,
    leftPath,
    leftItems,
    leftLoading,
    leftCanary,
    leftSelected,
    leftUploading,
    rightRemoteName,
    rightPath,
    rightItems,
    rightLoading,
    rightCanary,
    rightSelected,
    rightUploading,
    leftIsLocal,
    rightIsLocal,
    leftRemote,
    rightRemote,
    swapPanes,
    refreshPane,
    refreshAll,
    handleRemoteUpdate,
    handleNavigate,
    getRelativeItemPath,
    upDirectory,
    touchCanary,
    handleUploadFiles,
  }
}
