import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useTelemetryStore } from './telemetry'
import { useQueueStore } from './queue'
import { useBandwidthStore } from './bandwidth'

export const useAppStore = defineStore('app', () => {
  const activeTab = ref<string>('fleet')
  const isPollingPaused = ref<boolean>(false)
  const quickCreateOpen = ref<boolean>(false)
  const shortcutsModalOpen = ref<boolean>(false)

  function switchTab(tab: string) {
    activeTab.value = tab
  }

  function toggleQuickCreate() {
    quickCreateOpen.value = !quickCreateOpen.value
  }

  function setQuickCreate(val: boolean) {
    quickCreateOpen.value = val
  }

  function openShortcutsModal() {
    shortcutsModalOpen.value = true
  }

  function closeShortcutsModal() {
    shortcutsModalOpen.value = false
  }

  function initPolling() {
    if (typeof window === 'undefined') return

    const params = new URLSearchParams(window.location.search)
    const devParam = params.get('dev')
    const pollParam = params.get('poll')
    const isDev = devParam === 'true' || devParam === '1' || pollParam === 'false' || pollParam === '0'
    const storedPause = localStorage.getItem('bisync_polling_paused')

    if (isDev || storedPause === 'true') {
      isPollingPaused.value = true
    } else {
      isPollingPaused.value = false
    }

    // Power management & battery saver: suspend polling on background tab
    document.addEventListener('visibilitychange', () => {
      const telemetryStore = useTelemetryStore()
      const queueStore = useQueueStore()
      const bandwidthStore = useBandwidthStore()

      if (document.hidden) {
        telemetryStore.stopPolling()
        queueStore.stopPolling()
        bandwidthStore.stopPolling()
      } else if (!isPollingPaused.value) {
        telemetryStore.startPolling(3000)
        queueStore.startPolling(3000)
        bandwidthStore.startPolling(5000)
      }
    })
  }

  function togglePolling(): boolean {
    isPollingPaused.value = !isPollingPaused.value
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('bisync_polling_paused', String(isPollingPaused.value))
    }

    const telemetryStore = useTelemetryStore()
    const queueStore = useQueueStore()
    const bandwidthStore = useBandwidthStore()

    if (isPollingPaused.value) {
      telemetryStore.stopPolling()
      queueStore.stopPolling()
      bandwidthStore.stopPolling()
    } else {
      telemetryStore.startPolling(3000)
      queueStore.startPolling(3000)
      bandwidthStore.startPolling(5000)
    }

    return isPollingPaused.value
  }

  // Backwards compatibility bridge for e2e tests & window access
  if (typeof window !== 'undefined') {
    const win = window as any
    win.initPolling = initPolling
    win.togglePolling = togglePolling
    Object.defineProperty(win, 'quickCreateOpen', {
      get: () => quickCreateOpen.value,
      set: (val: boolean) => { quickCreateOpen.value = val },
      configurable: true,
    })
    Object.defineProperty(win, 'isPollingPaused', {
      get: () => isPollingPaused.value,
      set: (val: boolean) => { isPollingPaused.value = val },
      configurable: true,
    })
  }

  return {
    activeTab,
    isPollingPaused,
    quickCreateOpen,
    shortcutsModalOpen,
    switchTab,
    toggleQuickCreate,
    setQuickCreate,
    openShortcutsModal,
    closeShortcutsModal,
    initPolling,
    togglePolling,
  }
})
