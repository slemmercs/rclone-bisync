<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { api } from '../api/client'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import { useProfilesStore } from '../stores/profiles'
import type { BrowseItem } from '../types/models'

const profilesStore = useProfilesStore()

const localPath = ref<string>('')
const localItems = ref<BrowseItem[]>([])
const localLoading = ref<boolean>(false)
const localCanary = ref<boolean>(false)
const localSelected = ref<BrowseItem | null>(null)

const remoteName = ref<string>('local')
const remotePath = ref<string>('')
const remoteItems = ref<BrowseItem[]>([])
const remoteLoading = ref<boolean>(false)
const remoteCanary = ref<boolean>(false)
const remoteSelected = ref<BrowseItem | null>(null)

const activeProfile = computed(() => profilesStore.activeProfile)

watch(
  () => activeProfile.value,
  (profile) => {
    if (profile) {
      localPath.value = profile.local_path || ''
      remoteName.value = profile.remote || 'local'
      remotePath.value = profile.remote_path || ''
      refreshLocal()
      refreshRemote()
    }
  },
  { immediate: true }
)

async function fetchDirectory(remote: string, path: string) {
  const profileId = activeProfile.value?.id
  return await api.post('/api/browse', { remote, path, profile_id: profileId })
}

async function refreshLocal() {
  localLoading.value = true
  localSelected.value = null
  try {
    const data = await fetchDirectory('local', localPath.value)
    localItems.value = (data.items || []) as BrowseItem[]
    localCanary.value = !!data.has_canary
  } catch (e) {
    localItems.value = []
  } finally {
    localLoading.value = false
  }
}

async function refreshRemote() {
  remoteLoading.value = true
  remoteSelected.value = null
  try {
    const data = await fetchDirectory(remoteName.value, remotePath.value)
    remoteItems.value = (data.items || []) as BrowseItem[]
    remoteCanary.value = !!data.has_canary
  } catch (e) {
    remoteItems.value = []
  } finally {
    remoteLoading.value = false
  }
}

function navigateLocal(item: BrowseItem) {
  if (item.IsDir) {
    const next = localPath.value.replace(/\/+$/, '') + '/' + item.Name
    localPath.value = next
    refreshLocal()
  } else {
    localSelected.value = item
  }
}

function navigateRemote(item: BrowseItem) {
  if (item.IsDir) {
    const next = remotePath.value ? remotePath.value.replace(/\/+$/, '') + '/' + item.Name : item.Name
    remotePath.value = next
    refreshRemote()
  } else {
    remoteSelected.value = item
  }
}

function upLocal() {
  const parts = localPath.value.replace(/\/+$/, '').split('/')
  if (parts.length > 1) {
    parts.pop()
    localPath.value = parts.join('/') || '/'
    refreshLocal()
  }
}

function upRemote() {
  const parts = remotePath.value.replace(/\/+$/, '').split('/')
  if (parts.length > 1) {
    parts.pop()
    remotePath.value = parts.join('/')
    refreshRemote()
  } else {
    remotePath.value = ''
    refreshRemote()
  }
}

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-base font-semibold text-carbon-50 tracking-tight">Dual-Pane Storage Explorer</h2>
        <p class="text-xs text-carbon-400">Direct side-by-side inspection of local files vs remote cloud targets with safety canary checks.</p>
      </div>
    </div>
    <!-- Dual Explorer Splitter -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg overflow-hidden shadow-sm min-h-[560px]">
      <Splitter class="min-h-[560px] bg-carbon-900 text-carbon-100 border-none">
        <!-- Local Storage Panel -->
        <SplitterPanel :size="50" :minSize="30" class="p-3 flex flex-col border-r border-carbon-700/80">
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-700">
            <div class="flex items-center gap-2">
              <i class="pi pi-desktop text-carbon-400"></i>
              <span class="text-xs font-bold text-carbon-50 uppercase tracking-wide">Local Storage</span>
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="localCanary ? 'bg-emerald-950/70 text-emerald-400 border border-emerald-600/40' : 'bg-carbon-800 text-carbon-500'"
              >
                <i :class="localCanary ? 'pi pi-check' : 'pi pi-info-circle'" class="text-[9px]"></i>
                {{ localCanary ? 'Canary OK' : 'No Canary' }}
              </span>
            </div>
            <div class="flex items-center gap-1">
              <button @click="upLocal" class="p-1 rounded bg-carbon-800 text-carbon-300 hover:text-white" title="Up">
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button @click="refreshLocal" class="p-1 rounded bg-carbon-800 text-carbon-300 hover:text-white" title="Refresh">
                <i :class="localLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <div class="mb-2 bg-carbon-950 px-2 py-1 rounded border border-carbon-800 text-xs font-mono text-carbon-400 truncate">
            {{ localPath || '/' }}
          </div>

          <!-- Local Items -->
          <div class="flex-1 overflow-y-auto max-h-[440px] divide-y divide-carbon-800/60 text-xs">
            <div
              v-for="item in localItems"
              :key="item.Name"
              @click="navigateLocal(item)"
              class="flex items-center justify-between px-2.5 py-1.5 cursor-pointer select-none transition-colors"
              :class="localSelected?.Name === item.Name ? 'bg-carbon-750 text-electric-400 font-semibold' : 'hover:bg-carbon-800/80 text-carbon-200'"
            >
              <div class="flex items-center gap-2 truncate">
                <i :class="item.IsDir ? 'pi pi-folder text-electric-400' : 'pi pi-file text-carbon-400'" class="text-xs"></i>
                <span class="truncate">{{ item.Name }}</span>
              </div>
              <span class="text-[11px] font-mono text-carbon-500 shrink-0">{{ item.IsDir ? '--' : formatBytes(item.Size) }}</span>
            </div>
            <div v-if="localItems.length === 0 && !localLoading" class="p-8 text-center text-xs text-carbon-500">
              Empty directory or path inaccessible
            </div>
          </div>
        </SplitterPanel>

        <!-- Remote Storage Panel -->
        <SplitterPanel :size="50" :minSize="30" class="p-3 flex flex-col">
          <div class="flex items-center justify-between gap-2 mb-2 pb-2 border-b border-carbon-700">
            <div class="flex items-center gap-2">
              <i class="pi pi-cloud text-electric-400"></i>
              <span class="text-xs font-bold text-carbon-50 uppercase tracking-wide">Remote: {{ remoteName }}</span>
              <span
                class="text-[10px] font-mono px-2 py-0.5 rounded flex items-center gap-1"
                :class="remoteCanary ? 'bg-emerald-950/70 text-emerald-400 border border-emerald-600/40' : 'bg-carbon-800 text-carbon-500'"
              >
                <i :class="remoteCanary ? 'pi pi-check' : 'pi pi-info-circle'" class="text-[9px]"></i>
                {{ remoteCanary ? 'Canary OK' : 'No Canary' }}
              </span>
            </div>
            <div class="flex items-center gap-1">
              <button @click="upRemote" class="p-1 rounded bg-carbon-800 text-carbon-300 hover:text-white" title="Up">
                <i class="pi pi-arrow-up text-xs"></i>
              </button>
              <button @click="refreshRemote" class="p-1 rounded bg-carbon-800 text-carbon-300 hover:text-white" title="Refresh">
                <i :class="remoteLoading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
              </button>
            </div>
          </div>

          <div class="mb-2 bg-carbon-950 px-2 py-1 rounded border border-carbon-800 text-xs font-mono text-carbon-400 truncate">
            {{ remoteName }}:{{ remotePath ? `/${remotePath}` : '' }}
          </div>

          <!-- Remote Items -->
          <div class="flex-1 overflow-y-auto max-h-[440px] divide-y divide-carbon-800/60 text-xs">
            <div
              v-for="item in remoteItems"
              :key="item.Name"
              @click="navigateRemote(item)"
              class="flex items-center justify-between px-2.5 py-1.5 cursor-pointer select-none transition-colors"
              :class="remoteSelected?.Name === item.Name ? 'bg-carbon-750 text-electric-400 font-semibold' : 'hover:bg-carbon-800/80 text-carbon-200'"
            >
              <div class="flex items-center gap-2 truncate">
                <i :class="item.IsDir ? 'pi pi-folder text-electric-400' : 'pi pi-file text-carbon-400'" class="text-xs"></i>
                <span class="truncate">{{ item.Name }}</span>
              </div>
              <span class="text-[11px] font-mono text-carbon-500 shrink-0">{{ item.IsDir ? '--' : formatBytes(item.Size) }}</span>
            </div>
            <div v-if="remoteItems.length === 0 && !remoteLoading" class="p-8 text-center text-xs text-carbon-500">
              Empty directory or path inaccessible
            </div>
          </div>
        </SplitterPanel>
      </Splitter>
    </div>
  </div>
</template>
