import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ActiveTransfer, BandwidthLimitResponse, CoreStatsResponse } from '../types/models'
import { formatBytes, formatSpeed } from '../composables/useFormatters'

export const useBandwidthStore = defineStore('bandwidth', () => {
  const bytesPerSecond = ref<number>(0)
  const rateFormatted = ref<string>('off')
  const currentLimit = ref<string>('off')
  const customInput = ref<string>('')
  const presets = ref<string[]>(['off', '500k', '2M', '10M', '50M'])

  const speed = ref<number>(0)
  const bytes = ref<number>(0)
  const transfers = ref<number>(0)
  const checks = ref<number>(0)
  const errors = ref<number>(0)
  const fatalErrors = ref<boolean>(false)
  const transferring = ref<ActiveTransfer[]>([])

  const popoverOpen = ref<boolean>(false)
  const loading = ref<boolean>(false)
  const updating = ref<boolean>(false)
  const lastError = ref<string | null>(null)

  let pollTimer: ReturnType<typeof setInterval> | null = null

  async function fetchBandwidthLimit(): Promise<BandwidthLimitResponse | null> {
    try {
      const res = await fetch('/api/core/bwlimit')
      if (!res.ok) throw new Error(`HTTP error ${res.status}`)
      const data: BandwidthLimitResponse = await res.json()
      bytesPerSecond.value = data.bytes_per_second || 0
      rateFormatted.value = data.rate_formatted || (data.bytes_per_second > 0 ? formatSpeed(data.bytes_per_second) : 'off')
      currentLimit.value = rateFormatted.value
      lastError.value = null
      return data
    } catch (err: any) {
      lastError.value = err?.message || 'Failed to fetch bandwidth limit'
      return null
    }
  }

  async function fetchCoreStats(): Promise<CoreStatsResponse | null> {
    try {
      const res = await fetch('/api/core/stats')
      if (!res.ok) throw new Error(`HTTP error ${res.status}`)
      const data: CoreStatsResponse = await res.json()
      speed.value = data.speed || 0
      bytes.value = data.bytes || 0
      transfers.value = data.transfers || 0
      checks.value = data.checks || 0
      errors.value = data.errors || 0
      fatalErrors.value = Boolean(data.fatal_errors)
      transferring.value = Array.isArray(data.transferring) ? data.transferring : []
      return data
    } catch {
      return null
    }
  }

  async function setBandwidthLimit(rate: string): Promise<boolean> {
    if (!rate) return false
    updating.value = true
    lastError.value = null
    try {
      const res = await fetch('/api/core/bwlimit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ rate }),
      })
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || `HTTP error ${res.status}`)
      }
      const data: BandwidthLimitResponse = await res.json()
      rateFormatted.value = data.rate_formatted || rate
      currentLimit.value = data.rate_formatted || rate
      bytesPerSecond.value = data.bytes_per_second || 0
      return true
    } catch (err: any) {
      lastError.value = err?.message || 'Failed to set bandwidth limit'
      throw err
    } finally {
      updating.value = false
    }
  }

  async function applyCustomBandwidthLimit(): Promise<boolean> {
    const rate = customInput.value.trim()
    if (!rate) return false
    const success = await setBandwidthLimit(rate)
    if (success) customInput.value = ''
    return success
  }

  function togglePopover() {
    popoverOpen.value = !popoverOpen.value
    if (popoverOpen.value) {
      fetchBandwidthLimit()
      fetchCoreStats()
    }
  }

  function openPopover() {
    popoverOpen.value = true
    fetchBandwidthLimit()
    fetchCoreStats()
  }

  function closePopover() {
    popoverOpen.value = false
  }

  function startPolling(intervalMs: number = 3000) {
    if (pollTimer) return
    fetchBandwidthLimit()
    fetchCoreStats()
    pollTimer = setInterval(() => {
      fetchBandwidthLimit()
      fetchCoreStats()
    }, intervalMs)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  if (typeof window !== 'undefined') {
    const win = window as any
    win.BandwidthModule = win.BandwidthModule || {
      bandwidth: {
        get speed() { return speed.value },
        get bytes() { return bytes.value },
        get transfers() { return transfers.value },
        get checks() { return checks.value },
        get errors() { return errors.value },
        get fatalErrors() { return fatalErrors.value },
        get transferring() { return transferring.value },
        get currentLimit() { return currentLimit.value },
        get bytesPerSecond() { return bytesPerSecond.value },
        get rateFormatted() { return rateFormatted.value },
        get popoverOpen() { return popoverOpen.value },
        get customInput() { return customInput.value },
        set customInput(val: string) { customInput.value = val },
        get loading() { return loading.value },
        get updating() { return updating.value },
        get presets() { return presets.value },
      },
      isBandwidthUpdating: () => updating.value,
      isBandwidthPopoverOpen: () => popoverOpen.value,
      toggleBandwidthPopover: togglePopover,
      closeBandwidthPopover: closePopover,
      fetchBandwidthLimit,
      fetchCoreStats,
      setBandwidthLimit,
      applyCustomBandwidthLimit,
      formatSpeed,
      formatBytes,
    }
  }

  return {
    bytesPerSecond,
    rateFormatted,
    currentLimit,
    customInput,
    presets,
    speed,
    bytes,
    transfers,
    checks,
    errors,
    fatalErrors,
    transferring,
    popoverOpen,
    loading,
    updating,
    lastError,
    fetchBandwidthLimit,
    fetchCoreStats,
    setBandwidthLimit,
    applyCustomBandwidthLimit,
    togglePopover,
    openPopover,
    closePopover,
    startPolling,
    stopPolling,
  }
})
