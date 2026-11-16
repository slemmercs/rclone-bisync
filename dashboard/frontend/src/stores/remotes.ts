import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'
import type { RemotesResponse } from '../types/models'

export const useRemotesStore = defineStore('remotes', () => {
  const remotes = ref<Array<Record<string, any>>>([])
  const summary = ref<Record<string, any>>({})
  const count = ref<number>(0)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  async function fetchRemotes(refresh: boolean = false) {
    loading.value = true
    error.value = null
    try {
      const url = refresh ? '/api/remotes?refresh=true' : '/api/remotes'
      const data = await api.get<RemotesResponse>(url)
      remotes.value = data.remotes || []
      summary.value = data.summary || {}
      count.value = data.count || remotes.value.length
    } catch (e: any) {
      error.value = e.message || 'Error fetching remotes'
    } finally {
      loading.value = false
    }
  }

  async function checkAuth(remoteName?: string) {
    loading.value = true
    try {
      return await api.post('/api/remotes/auth/check', { remote: remoteName })
    } finally {
      loading.value = false
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
    loading,
    error,
    fetchRemotes,
    checkAuth,
    createRemote,
    deleteRemote,
  }
})
