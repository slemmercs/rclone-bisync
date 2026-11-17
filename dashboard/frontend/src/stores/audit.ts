import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'
import type {
  AuditRecord,
  AuditVerification,
  AuditRecordsResponse,
  AuditResetResponse,
} from '../types/models'

export const useAuditStore = defineStore('audit', () => {
  const records = ref<AuditRecord[]>([])
  const total = ref<number>(0)
  const page = ref<number>(1)
  const limit = ref<number>(50)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  const filterAction = ref<string>('')
  const filterProfile = ref<string>('')
  const filterActor = ref<string>('')
  const searchQuery = ref<string>('')

  const verification = ref<AuditVerification>({
    valid: true,
    total_records: 0,
    checked_at: null,
    head_hash: '',
    error: null,
    checking: false,
  })

  const selectedRecord = ref<AuditRecord | null>(null)
  const detailsModalOpen = ref<boolean>(false)
  const resetModalOpen = ref<boolean>(false)
  const copiedHash = ref<boolean>(false)

  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / limit.value)))

  async function fetchAuditRecords() {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      params.set('limit', String(limit.value))
      params.set('page', String(page.value))
      if (filterAction.value) params.set('action', filterAction.value)
      if (filterProfile.value) params.set('profile_id', filterProfile.value)
      if (filterActor.value) params.set('actor', filterActor.value)

      const res = await api.get<AuditRecordsResponse>(`/api/audit/records?${params.toString()}`)
      records.value = res.records || []
      total.value = res.total || 0
    } catch (err: any) {
      error.value = err.message || 'Failed to query audit ledger'
    } finally {
      loading.value = false
    }
  }

  async function verifyAuditLedger() {
    verification.value.checking = true
    try {
      const res = await api.get<AuditVerification>('/api/audit/verify')
      verification.value.valid = !!res.valid
      verification.value.total_records = res.total_records || 0
      verification.value.head_hash = res.head_hash || ''
      verification.value.checked_at = res.checked_at
        ? new Date(res.checked_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
        : new Date().toLocaleTimeString()
      verification.value.error = res.error || null
    } catch (err: any) {
      verification.value.valid = false
      verification.value.error = err.message || 'Verification check error'
    } finally {
      verification.value.checking = false
    }
  }

  async function resetAuditLedger(): Promise<AuditResetResponse> {
    try {
      const res = await api.post<AuditResetResponse>('/api/audit/reset')
      page.value = 1
      await Promise.all([fetchAuditRecords(), verifyAuditLedger()])
      return res
    } catch (err: any) {
      throw new Error(err.message || 'Failed to reset audit ledger')
    }
  }

  function copyHeadHash(hash: string) {
    if (!hash) return
    if (navigator?.clipboard?.writeText) {
      navigator.clipboard.writeText(hash).then(() => {
        copiedHash.value = true
        setTimeout(() => {
          copiedHash.value = false
        }, 2000)
      })
    }
  }

  function inspectRecord(rec: AuditRecord) {
    selectedRecord.value = rec
    detailsModalOpen.value = true
  }

  function closeDetails() {
    detailsModalOpen.value = false
    selectedRecord.value = null
  }

  function changePage(newPage: number) {
    if (newPage < 1 || newPage > totalPages.value) return
    page.value = newPage
    fetchAuditRecords()
  }

  function applyFilters() {
    page.value = 1
    fetchAuditRecords()
  }

  function resetFilters() {
    filterAction.value = ''
    filterProfile.value = ''
    filterActor.value = ''
    searchQuery.value = ''
    page.value = 1
    fetchAuditRecords()
  }

  return {
    records,
    total,
    page,
    limit,
    totalPages,
    loading,
    error,
    filterAction,
    filterProfile,
    filterActor,
    searchQuery,
    verification,
    selectedRecord,
    detailsModalOpen,
    resetModalOpen,
    copiedHash,
    fetchAuditRecords,
    verifyAuditLedger,
    resetAuditLedger,
    copyHeadHash,
    inspectRecord,
    closeDetails,
    changePage,
    applyFilters,
    resetFilters,
  }
})
