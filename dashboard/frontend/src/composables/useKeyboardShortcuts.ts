import { onMounted, onUnmounted } from 'vue'
import { useAppStore } from '../stores/app'
import { useLogsStore } from '../stores/logs'
import { useHelpStore } from '../stores/help'
import { useSettingsStore } from '../stores/settings'
import { useToastNotification } from './useToastNotification'

export interface KeyboardShortcutHandlers {
  onOpenProfiles?: () => void
  onOpenRemoteWizard?: () => void
  onOpenReview?: () => void
}

export function useKeyboardShortcuts(handlers?: KeyboardShortcutHandlers) {
  const appStore = useAppStore()
  const logsStore = useLogsStore()
  const helpStore = useHelpStore()
  const settingsStore = useSettingsStore()
  const toast = useToastNotification()

  function handleKeyDown(e: KeyboardEvent) {
    const target = e.target as HTMLElement | null
    const tag = target?.tagName?.toLowerCase() || ''
    const isEditing = tag === 'input' || tag === 'textarea' || tag === 'select' || !!target?.isContentEditable

    // Alt + P: Toggle telemetry polling (global, even inside inputs)
    if (e.altKey && (e.key === 'p' || e.key === 'P')) {
      e.preventDefault()
      const paused = appStore.togglePolling()
      if (paused) {
        toast.info('Telemetry Polling Paused', 'Press Alt+P or click indicator to resume live telemetry.')
      } else {
        toast.success('Telemetry Polling Resumed', 'Live telemetry and scheduler active.')
      }
      return
    }

    // Alt + R: Open Conflict Review
    if (e.altKey && (e.key === 'r' || e.key === 'R')) {
      e.preventDefault()
      handlers?.onOpenReview?.()
      return
    }

    // Ctrl+K / Cmd+K: Command / Quick-Create menu
    if ((e.ctrlKey || e.metaKey) && (e.key === 'k' || e.key === 'K')) {
      e.preventDefault()
      appStore.toggleQuickCreate()
      return
    }

    // Escape: Close top-level overlays
    if (e.key === 'Escape') {
      if (appStore.quickCreateOpen) {
        appStore.setQuickCreate(false)
        return
      }
      if (appStore.shortcutsModalOpen) {
        appStore.closeShortcutsModal()
        return
      }
      if (settingsStore.showSystemHealthModal) {
        settingsStore.closeSystemHealthModal()
        return
      }
      if (settingsStore.showGlobalSettingsModal) {
        settingsStore.closeGlobalSettingsModal()
        return
      }
      if (settingsStore.showMcpModal) {
        settingsStore.closeMcpModal()
        return
      }
      if (helpStore.isOpen) {
        helpStore.closeDrawer()
        return
      }
      return
    }

    // Ignore single-character hotkeys while typing in text inputs or form controls
    if (isEditing) return

    // ? or F1: Open Help & Diagnostics Drawer
    if (e.key === '?' || e.key === 'F1') {
      e.preventDefault()
      helpStore.openDrawer()
      return
    }

    // C / c: Quick Create Menu
    if ((e.key === 'c' || e.key === 'C') && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault()
      appStore.toggleQuickCreate()
      return
    }

    // N / n: New Sync Profile
    if ((e.key === 'n' || e.key === 'N') && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault()
      appStore.setQuickCreate(false)
      handlers?.onOpenProfiles?.()
      return
    }

    // R / r: Connect Storage Remote
    if ((e.key === 'r' || e.key === 'R') && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault()
      appStore.setQuickCreate(false)
      handlers?.onOpenRemoteWizard?.()
      return
    }

    // L / l: Toggle Log Dock
    if ((e.key === 'l' || e.key === 'L') && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault()
      logsStore.toggleDock()
      return
    }

    // 1-5: Workspace View Navigation
    if (['1', '2', '3', '4', '5'].includes(e.key) && !e.ctrlKey && !e.metaKey && !e.altKey) {
      const tabMap: Record<string, string> = {
        '1': 'fleet',
        '2': 'telemetry',
        '3': 'remotes',
        '4': 'explorer',
        '5': 'audit',
      }
      const targetTab = tabMap[e.key]
      if (targetTab) {
        e.preventDefault()
        appStore.switchTab(targetTab)
      }
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeyDown)
  })

  return {
    handleKeyDown,
  }
}
