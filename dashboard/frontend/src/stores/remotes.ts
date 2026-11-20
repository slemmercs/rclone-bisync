import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'
import type {
  RemotesResponse,
  AuthStatusResponse,
  RemoteAuthInfo,
  AuthHealthSummary,
  OAuthAuditEvent,
  AuditEventsResponse,
  ReconnectResponse,
  ValidateResponse,
} from '../types/models'

export const useRemotesStore = defineStore('remotes', () => {
  const remotes = ref<Array<Record<string, any>>>([])
  const summary = ref<Record<string, any>>({})
  const count = ref<number>(0)
  const authSummary = ref<AuthHealthSummary | null>(null)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  // Per-remote action states
  const reconnecting = ref<Record<string, boolean>>({})
  const validating = ref<Record<string, boolean>>({})
  const validated = ref<Record<string, boolean>>({})

  // Reconnect Modal State
  const reconnectModal = ref({
    show: false,
    remote: '',
    loading: false,
    authUrl: null as string | null,
    msg: null as string | null,
    error: null as string | null,
  })

  // Migration / OAuth App Config Modal State
  const migrationModal = ref({
    show: false,
    remote: '',
    type: '',
    typeLabel: '',
    clientId: '',
    clientSecret: '',
    showSecret: false,
    loading: false,
    error: null as string | null,
    success: null as string | null,
  })

  // Auth Audit Modal State
  const auditModal = ref({
    show: false,
    loading: false,
    events: [] as OAuthAuditEvent[],
  })

  async function fetchRemotes(refresh: boolean = false) {
    loading.value = true
    error.value = null
    try {
      const remotesUrl = refresh ? '/api/remotes?refresh=true' : '/api/remotes'
      const authUrl = refresh ? '/api/remotes/auth?refresh=true' : '/api/remotes/auth'

      const [remotesRes, authRes] = await Promise.allSettled([
        api.get<RemotesResponse>(remotesUrl),
        api.get<AuthStatusResponse>(authUrl),
      ])

      let rawRemotes: Array<Record<string, any>> = []
      if (remotesRes.status === 'fulfilled') {
        rawRemotes = remotesRes.value.remotes || []
        summary.value = remotesRes.value.summary || {}
        count.value = remotesRes.value.count || rawRemotes.length
      }

      const authMap: Record<string, RemoteAuthInfo> = {}
      if (authRes.status === 'fulfilled' && authRes.value.remotes) {
        authSummary.value = authRes.value.summary || null
        for (const a of authRes.value.remotes) {
          authMap[a.name] = a
        }
      }

      remotes.value = rawRemotes.map((r) => ({
        ...r,
        auth: authMap[r.name] || r.auth || null,
      }))
    } catch (e: any) {
      error.value = e.message || 'Error fetching remotes'
    } finally {
      loading.value = false
    }
  }

  async function checkAuth(remoteName?: string) {
    loading.value = true
    try {
      const url = remoteName
        ? `/api/remotes/auth/check?remote=${encodeURIComponent(remoteName)}`
        : '/api/remotes/auth/check'
      const res = await api.post<AuthStatusResponse>(url)
      await fetchRemotes(false)
      return res
    } finally {
      loading.value = false
    }
  }

  async function validateRemote(name: string): Promise<ValidateResponse | null> {
    if (!name || validating.value[name]) return null

    validating.value = { ...validating.value, [name]: true }
    validated.value = { ...validated.value, [name]: false }
    const startTime = Date.now()

    try {
      const res = await api.post<ValidateResponse>(`/api/remotes/${encodeURIComponent(name)}/validate`)

      const elapsed = Date.now() - startTime
      if (elapsed < 600) {
        await new Promise((resolve) => setTimeout(resolve, 600 - elapsed))
      }

      if (res && res.valid) {
        validated.value = { ...validated.value, [name]: true }
        setTimeout(() => {
          validated.value = { ...validated.value, [name]: false }
        }, 4000)
      }

      await fetchRemotes(false)
      return res
    } catch (err: any) {
      return {
        status: 'error',
        remote: name,
        valid: false,
        error: err.message || 'Validation request failed',
      }
    } finally {
      validating.value = { ...validating.value, [name]: false }
    }
  }

  async function reconnectRemote(name: string) {
    reconnecting.value = { ...reconnecting.value, [name]: true }
    reconnectModal.value = {
      show: true,
      remote: name,
      loading: true,
      authUrl: null,
      msg: 'Initiating desktop OAuth reconnect flow...',
      error: null,
    }

    try {
      const data = await api.post<ReconnectResponse>(`/api/remotes/${encodeURIComponent(name)}/reconnect`)
      if (data && data.status === 'success') {
        reconnectModal.value.msg = data.msg || `Re-authenticated "${name}" successfully.`
        reconnectModal.value.authUrl = data.auth_url || null
        await fetchRemotes(false)
        if (auditModal.value.show) await viewAuthAudit()
      } else {
        reconnectModal.value.error = data?.error || data?.msg || 'Re-authentication failed.'
        reconnectModal.value.authUrl = data?.auth_url || null
      }
    } catch (err: any) {
      reconnectModal.value.error = 'Network error: ' + (err.message || 'Unknown error')
    } finally {
      reconnecting.value = { ...reconnecting.value, [name]: false }
      reconnectModal.value.loading = false
    }
  }

  async function cancelReconnect(name: string) {
    try {
      await api.post(`/api/remotes/${encodeURIComponent(name)}/reconnect/cancel`)
    } catch (e) {
      console.error(e)
    } finally {
      reconnecting.value = { ...reconnecting.value, [name]: false }
      if (reconnectModal.value.remote === name) {
        reconnectModal.value.show = false
        reconnectModal.value.loading = false
      }
      await fetchRemotes(false)
      if (auditModal.value.show) await viewAuthAudit()
    }
  }

  async function stopOAuth() {
    try {
      await api.post('/api/remotes/auth/oauth-stop')
      if (reconnectModal.value.show) {
        reconnectModal.value.loading = false
        reconnectModal.value.error = 'Process stopped by user.'
      }
    } catch (err: any) {
      console.error('Error stopping OAuth:', err)
    }
  }

  function openMigrationWizard(remote: any) {
    migrationModal.value = {
      show: true,
      remote: remote.name,
      type: remote.type || '',
      typeLabel: remote.type_label || remote.name,
      clientId: remote.auth?.client_id || '',
      clientSecret: '',
      showSecret: false,
      loading: false,
      error: null,
      success: null,
    }
  }

  function closeMigrationWizard() {
    migrationModal.value.show = false
    migrationModal.value.error = null
    migrationModal.value.success = null
  }

  async function saveMigrationCredentials(): Promise<boolean> {
    if (!migrationModal.value.clientId || !migrationModal.value.clientId.trim()) {
      migrationModal.value.error = 'Client ID is required.'
      return false
    }

    migrationModal.value.loading = true
    migrationModal.value.error = null
    migrationModal.value.success = null

    try {
      const data = await api.post<{ status: string; error?: string }>(
        `/api/remotes/${encodeURIComponent(migrationModal.value.remote)}/credentials`,
        {
          client_id: migrationModal.value.clientId.trim(),
          client_secret: migrationModal.value.clientSecret ? migrationModal.value.clientSecret.trim() : '',
        }
      )

      if (data && data.status === 'success') {
        migrationModal.value.success = 'OAuth credentials successfully saved!'
        await fetchRemotes(false)
        setTimeout(() => {
          closeMigrationWizard()
        }, 1200)
        return true
      } else {
        migrationModal.value.error = data?.error || 'Failed to update OAuth credentials.'
        return false
      }
    } catch (err: any) {
      migrationModal.value.error = 'Network error: ' + (err.message || 'Unknown error')
      return false
    } finally {
      migrationModal.value.loading = false
    }
  }

  async function viewAuthAudit() {
    auditModal.value = { show: true, loading: true, events: [] }
    try {
      const data = await api.get<AuditEventsResponse>('/api/remotes/auth/audit?limit=50')
      auditModal.value.events = data.events || []
    } catch (err) {
      console.error('Failed to fetch auth audit:', err)
    } finally {
      auditModal.value.loading = false
    }
  }

  async function createRemote(payload: { name: string; type: string; parameters?: Record<string, any> }) {
    loading.value = true
    try {
      const res = await api.post('/api/remotes/create', payload)
      await fetchRemotes(true)
      return res
    } finally {
      loading.value = false
    }
  }

  async function deleteRemote(name: string) {
    loading.value = true
    try {
      await api.delete(`/api/remotes/${name}`)
      await fetchRemotes(true)
    } finally {
      loading.value = false
    }
  }

  return {
    remotes,
    summary,
    count,
    authSummary,
    loading,
    error,
    reconnecting,
    validating,
    validated,
    reconnectModal,
    migrationModal,
    auditModal,
    fetchRemotes,
    checkAuth,
    validateRemote,
    reconnectRemote,
    cancelReconnect,
    stopOAuth,
    openMigrationWizard,
    closeMigrationWizard,
    saveMigrationCredentials,
    viewAuthAudit,
    createRemote,
    deleteRemote,
  }
})
