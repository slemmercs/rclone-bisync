import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'
import type {
  GlobalConfigResponse,
  UpdateConfigRequest,
  ConfigEncryptionResponse,
  MCPStatus,
  GlobalDefaults,
  ConfigDaemonSettings,
  ConfigNotificationSettings,
} from '../types/models'

export const useSettingsStore = defineStore('settings', () => {
  // Modal visibility
  const showGlobalSettingsModal = ref<boolean>(false)
  const showSystemHealthModal = ref<boolean>(false)
  const showMcpModal = ref<boolean>(false)

  // Tab selections
  const globalSettingsTab = ref<'defaults' | 'engine' | 'notifications' | 'encryption'>('defaults')
  const systemHealthTab = ref<'daemon' | 'mcp' | 'connectivity'>('daemon')
  const mcpTab = ref<'overview' | 'config' | 'tools'>('overview')

  // Config data
  const globalDefaults = ref<GlobalDefaults>({
    max_delete: 0,
    canary_validation: true,
    conflict_resolve: 'newer',
    compare_mode: 'modtime',
    remove_empty_dirs: true,
    create_empty_dirs: true,
    lock_explorer_remote: true,
  })

  const daemonSettings = ref<ConfigDaemonSettings>({
    job_poll_initial_sleep_sec: 2,
    job_poll_max_sleep_sec: 10,
    job_timeout_seconds: 1800,
    lock_stale_timeout_seconds: 3600,
    rate_limit_backoff_hours: 1,
  })

  const notificationSettings = ref<ConfigNotificationSettings>({
    webhook_url: '',
    watchdog_ping_url: '',
  })

  const loadingConfig = ref<boolean>(false)
  const savingConfig = ref<boolean>(false)
  const configError = ref<string | null>(null)

  // Encryption data
  const encryption = ref<ConfigEncryptionResponse>({
    is_encrypted: false,
    has_daemon_passphrase: true,
    warning: '',
    config_file: '',
  })
  const loadingEncryption = ref<boolean>(false)
  const encryptionError = ref<string | null>(null)

  // MCP status data
  const mcpStatus = ref<MCPStatus>({
    status: 'offline',
    read_only: true,
  })
  const loadingMcp = ref<boolean>(false)

  function openGlobalSettingsModal(tab: 'defaults' | 'engine' | 'notifications' | 'encryption' = 'defaults') {
    globalSettingsTab.value = tab
    showGlobalSettingsModal.value = true
    fetchConfig()
    fetchEncryption()
  }

  function closeGlobalSettingsModal() {
    showGlobalSettingsModal.value = false
  }

  function openSystemHealthModal(tab: 'daemon' | 'mcp' | 'connectivity' = 'daemon') {
    systemHealthTab.value = tab
    showSystemHealthModal.value = true
    fetchMcpStatus()
  }

  function closeSystemHealthModal() {
    showSystemHealthModal.value = false
  }

  function openMcpModal(tab: 'overview' | 'config' | 'tools' = 'overview') {
    mcpTab.value = tab
    showMcpModal.value = true
    fetchMcpStatus()
  }

  function closeMcpModal() {
    showMcpModal.value = false
  }


  async function fetchConfig() {
    loadingConfig.value = true
    configError.value = null
    try {
      const data = await api.get<GlobalConfigResponse>('/api/config')
      if (data && data.status === 'success') {
        if (data.global_defaults) {
          globalDefaults.value = { ...data.global_defaults }
        }
        if (data.daemon) {
          daemonSettings.value = { ...data.daemon }
        }
        if (data.notifications) {
          notificationSettings.value = { ...data.notifications }
        }
      }
    } catch (e: any) {
      configError.value = e.message || 'Failed to load configuration'
    } finally {
      loadingConfig.value = false
    }
  }

  async function saveConfig(updates: UpdateConfigRequest): Promise<boolean> {
    savingConfig.value = true
    configError.value = null
    try {
      const res = await api.put<GlobalConfigResponse>('/api/config', updates)
      if (res && res.status === 'success') {
        if (res.global_defaults) {
          globalDefaults.value = { ...res.global_defaults }
        }
        if (res.daemon) {
          daemonSettings.value = { ...res.daemon }
        }
        if (res.notifications) {
          notificationSettings.value = { ...res.notifications }
        }
        return true
      }
      return false
    } catch (e: any) {
      configError.value = e.message || 'Failed to save configuration'
      throw e
    } finally {
      savingConfig.value = false
    }
  }

  async function fetchEncryption() {
    loadingEncryption.value = true
    encryptionError.value = null
    try {
      const data = await api.get<ConfigEncryptionResponse>('/api/config/encryption')
      if (data) {
        encryption.value = data
      }
    } catch (e: any) {
      encryptionError.value = e.message || 'Failed to query encryption status'
    } finally {
      loadingEncryption.value = false
    }
  }

  async function setPassword(password: string, oldPassword: string = ''): Promise<void> {
    loadingEncryption.value = true
    encryptionError.value = null
    try {
      await api.post('/api/config/encryption/set', { password, old_password: oldPassword })
      await fetchEncryption()
    } catch (e: any) {
      encryptionError.value = e.message || 'Failed to set encryption password'
      throw e
    } finally {
      loadingEncryption.value = false
    }
  }

  async function removePassword(password: string = ''): Promise<void> {
    loadingEncryption.value = true
    encryptionError.value = null
    try {
      await api.post('/api/config/encryption/remove', { password })
      await fetchEncryption()
    } catch (e: any) {
      encryptionError.value = e.message || 'Failed to remove encryption password'
      throw e
    } finally {
      loadingEncryption.value = false
    }
  }

  async function saveDaemonPassphrase(password: string): Promise<void> {
    loadingEncryption.value = true
    encryptionError.value = null
    try {
      await api.post('/api/config/encryption/save-passphrase', { password })
      await fetchEncryption()
    } catch (e: any) {
      encryptionError.value = e.message || 'Failed to save daemon passphrase'
      throw e
    } finally {
      loadingEncryption.value = false
    }
  }

  async function fetchMcpStatus() {
    loadingMcp.value = true
    try {
      const data = await api.get<MCPStatus>('/api/mcp/status')
      if (data) {
        mcpStatus.value = data
      }
    } catch (e: any) {
      console.warn('Failed to fetch MCP status:', e)
    } finally {
      loadingMcp.value = false
    }
  }

  return {
    showGlobalSettingsModal,
    showSystemHealthModal,
    showMcpModal,
    globalSettingsTab,
    systemHealthTab,
    mcpTab,
    globalDefaults,
    daemonSettings,
    notificationSettings,
    loadingConfig,
    savingConfig,
    configError,
    encryption,
    loadingEncryption,
    encryptionError,
    mcpStatus,
    loadingMcp,
    openGlobalSettingsModal,
    closeGlobalSettingsModal,
    openSystemHealthModal,
    closeSystemHealthModal,
    openMcpModal,
    closeMcpModal,
    fetchConfig,
    saveConfig,
    fetchEncryption,
    setPassword,
    removePassword,
    saveDaemonPassphrase,
    fetchMcpStatus,
  }
})
