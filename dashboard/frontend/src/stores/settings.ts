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
  ConfigLoggingSettings,
  ConfigCleanupSettings,
  ConfigTelemetrySettings,
  ConfigFeatureSettings,
} from '../types/models'

export const useSettingsStore = defineStore('settings', () => {
  // Modal visibility
  const showGlobalSettingsModal = ref<boolean>(false)
  const showSystemHealthModal = ref<boolean>(false)
  const showMcpModal = ref<boolean>(false)

  // Tab selections
  const globalSettingsTab = ref<'defaults' | 'flags' | 'engine' | 'logging' | 'telemetry' | 'notifications' | 'encryption'>('defaults')
  const systemHealthTab = ref<'daemon' | 'mcp' | 'connectivity' | 'poller'>('daemon')
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
    job_poll_backoff_multiplier: 1.5,
    job_timeout_seconds: 1800,
    lock_stale_timeout_seconds: 3600,
    rate_limit_backoff_hours: 1,
    rate_limit_retry_after_backoff: true,
    check_stale_locks: true,
  })

  const loggingSettings = ref<ConfigLoggingSettings>({
    max_log_size_mb: 5,
    log_backup_count: 5,
  })

  const cleanupSettings = ref<ConfigCleanupSettings>({
    files_to_cleanup: '',
  })

  const telemetrySettings = ref<ConfigTelemetrySettings>({
    enable_metrics: true,
    metrics_max_entries: 1000,
    metrics_max_backups: 5,
  })

  const featureSettings = ref<ConfigFeatureSettings>({
    debug_mode: false,
    mcp_read_only: false,
    stale_lock_detection: true,
    rate_limit_backoff: true,
    daemon_health_check: true,
    metrics_collection: true,
    smart_notifications: true,
    webhook_alerts: false,
  })

  const notificationSettings = ref<ConfigNotificationSettings>({
    app_name: 'Rclone Bisync',
    success_interval_min: 30,
    error_repeat_hours: 6,
    webhook_url: '',
    webhook_timeout_sec: 5,
    watchdog_ping_url: '',
    watchdog_timeout_sec: 5,
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

  function openSystemHealthModal(tab: 'daemon' | 'mcp' | 'connectivity' | 'poller' = 'daemon') {
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
        if (data.logging) {
          loggingSettings.value = { ...data.logging }
        }
        if (data.cleanup) {
          cleanupSettings.value = { ...data.cleanup }
        }
        if (data.telemetry) {
          telemetrySettings.value = { ...data.telemetry }
        }
        if (data.notifications) {
          notificationSettings.value = { ...data.notifications }
        }
        if (data.features) {
          featureSettings.value = { ...data.features }
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
        if (res.logging) {
          loggingSettings.value = { ...res.logging }
        }
        if (res.cleanup) {
          cleanupSettings.value = { ...res.cleanup }
        }
        if (res.telemetry) {
          telemetrySettings.value = { ...res.telemetry }
        }
        if (res.notifications) {
          notificationSettings.value = { ...res.notifications }
        }
        if (res.features) {
          featureSettings.value = { ...res.features }
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

  // Backwards compatibility bridge for e2e tests
  if (typeof window !== 'undefined') {
    const win = window as any
    win.openSystemHealthModal = openSystemHealthModal
    win.closeSystemHealthModal = closeSystemHealthModal
    win.systemHealthModal = {
      get show() { return showSystemHealthModal.value },
      set show(val: boolean) { showSystemHealthModal.value = val },
      get tab() { return systemHealthTab.value },
      set tab(val: 'daemon' | 'mcp' | 'connectivity' | 'poller') { systemHealthTab.value = val },
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
    loggingSettings,
    cleanupSettings,
    telemetrySettings,
    featureSettings,
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
