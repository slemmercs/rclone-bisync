<script setup lang="ts">
import { computed } from 'vue'
import { useRemotesStore } from '../../stores/remotes'
import { useDedupeStore } from '../../stores/dedupe'
import RemoteOverlayTree from './RemoteOverlayTree.vue'

interface Props {
  remote: Record<string, any>
  childrenByParent?: Record<string, any[]>
  activeRemoteName?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  childrenByParent: () => ({}),
  activeRemoteName: null
})

const emit = defineEmits<{
  (e: 'delete', name: string): void
}>()

const remotesStore = useRemotesStore()
const dedupeStore = useDedupeStore()

const directChildren = computed(() => {
  if (!props.childrenByParent || !props.remote.name) return []
  return props.childrenByParent[props.remote.name] || []
})

const hasDirectCrypt = computed(() => {
  if (props.remote.crypt_overlay) return true
  return directChildren.value.some((c: any) => (c.type || '').toLowerCase() === 'crypt')
})

const hasDirectHasher = computed(() => {
  if (props.remote.hasher_overlay) return true
  return directChildren.value.some((c: any) => (c.type || '').toLowerCase() === 'hasher')
})

const isBaseActive = computed(() => {
  if (props.remote.is_active_target) return true
  if (!props.activeRemoteName) return false
  return props.activeRemoteName === props.remote.name
})

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '-- GB'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function getQuotaPct(used?: number, total?: number): number {
  if (!used || !total || total <= 0) return 0
  const pct = (used / total) * 100
  return Math.min(Math.max(pct, 0), 100)
}

function getAuthBadge(status?: string): { text: string; dot: string; textClass: string } {
  switch (status) {
    case 'connected':
      return { text: 'Connected', dot: 'bg-emerald-400', textClass: 'text-emerald-400' }
    case 'migration_required':
      return { text: 'Migration Needed', dot: 'bg-amber-400 animate-pulse', textClass: 'text-amber-400' }
    case 'authorization_required':
      return { text: 'Auth Required', dot: 'bg-rose-400 animate-pulse', textClass: 'text-rose-400' }
    case 'configuration_required':
      return { text: 'Config Required', dot: 'bg-sky-400', textClass: 'text-sky-400' }
    case 'error':
      return { text: 'Auth Error', dot: 'bg-rose-500', textClass: 'text-rose-500' }
    default:
      return { text: 'Status Unknown', dot: 'bg-slate-400', textClass: 'text-slate-400' }
  }
}
</script>

<template>
  <div
    class="bg-carbon-surface border border-carbon-border hover:border-carbon-border/90 rounded-2xl p-5 space-y-3 transition-colors flex flex-col justify-between"
  >
    <div class="space-y-3">
      <div class="flex items-start justify-between">
        <div>
          <h4 class="font-bold text-white text-sm font-mono flex items-center gap-2">
            <span>{{ remote.name }}</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded uppercase bg-carbon-elevated text-slate-400 border border-carbon-border font-mono">
              {{ remote.type || 'remote' }}
            </span>
            <span
              v-if="(remote.type || '').toLowerCase() === 'hasher'"
              class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-amber-950/60 text-amber-300 border border-amber-800/60 font-mono flex items-center gap-1"
            >
              <i class="pi pi-bolt text-[8px]"></i> Accelerated
            </span>
            <span
              v-if="(remote.type || '').toLowerCase() === 'crypt'"
              class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-cyan-950/60 text-cyan-300 border border-cyan-800/60 font-mono flex items-center gap-1"
            >
              <i class="pi pi-shield text-[8px]"></i> Encrypted
            </span>
            <span
              v-if="directChildren.length === 0 && remote.crypt_overlay"
              class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-cyan-950/60 text-cyan-300 border border-cyan-800/60 font-mono flex items-center gap-1"
              :title="`Wrapped by encrypted overlay ${remote.crypt_overlay}`"
            >
              <i class="pi pi-shield text-[8px]"></i> Encrypted: {{ remote.crypt_overlay }}
            </span>
            <span
              v-if="directChildren.length === 0 && remote.hasher_overlay"
              class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-amber-950/60 text-amber-300 border border-amber-800/60 font-mono flex items-center gap-1"
              :title="`Wrapped by hasher overlay ${remote.hasher_overlay}`"
            >
              <i class="pi pi-bolt text-[8px]"></i> Accelerated: {{ remote.hasher_overlay }}
            </span>
            <span
              v-if="isBaseActive"
              class="text-[9px] uppercase font-bold px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-800/60 font-mono"
              title="Target endpoint of active sync profile"
            >
              Active
            </span>
          </h4>
          <div class="flex items-center gap-2 text-[11px] text-slate-400 font-mono mt-1">
            <span v-if="remote.underlying_path || remote.underlying_remote" class="text-cyan-400/90 font-medium flex items-center gap-1">
              <i class="pi pi-link text-[10px]"></i> Wraps: {{ remote.underlying_path || (remote.underlying_remote + ':') }}
            </span>
            <span v-else>{{ remote.endpoint || remote.root || 'Configured storage backend' }}</span>
            <template v-if="remote.auth">
              <span>•</span>
              <span class="inline-flex items-center gap-1" :class="getAuthBadge(remote.auth.status).textClass">
                <span class="w-1.5 h-1.5 rounded-full" :class="getAuthBadge(remote.auth.status).dot"></span>
                <span>{{ getAuthBadge(remote.auth.status).text }}</span>
              </span>
            </template>
          </div>
        </div>
        <span
          class="text-[10px] font-mono px-2 py-0.5 rounded-full font-bold uppercase"
          :class="remote.status === 'error' ? 'bg-rose-500/10 text-rose-400 border border-rose-500/30' : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'"
        >
          {{ remote.status === 'error' ? 'ERROR' : 'CONNECTED' }}
        </span>
      </div>

      <!-- Migration Required Warning Banner -->
      <div
        v-if="remote.auth && remote.auth.status === 'migration_required'"
        class="p-2.5 rounded-xl bg-amber-950/40 border border-amber-800/50 flex items-center justify-between gap-2 text-xs text-amber-200 font-mono"
      >
        <div class="flex items-center gap-1.5 text-[11px]">
          <i class="pi pi-exclamation-triangle text-amber-400 text-xs shrink-0"></i>
          <span>Shared Client ID. Personal credentials recommended.</span>
        </div>
        <button
          type="button"
          @click="remotesStore.openMigrationWizard(remote)"
          class="px-2 py-0.5 text-[10px] font-bold bg-amber-500 hover:bg-amber-400 text-slate-950 rounded transition-colors shrink-0 cursor-pointer"
        >
          Configure
        </button>
      </div>

      <!-- Storage Quota Section -->
      <div
        v-if="remote.about && (remote.about.total || remote.about.used)"
        class="bg-carbon-bg p-3 rounded-xl border border-carbon-border space-y-2"
      >
        <div class="flex justify-between items-baseline text-xs font-mono">
          <span class="text-slate-400 text-[11px]">Storage Allocation</span>
          <span class="text-white font-semibold text-[11px]">
            {{ formatBytes(remote.about.used) }} / {{ formatBytes(remote.about.total) }}
          </span>
        </div>

        <!-- Multi-segment Quota Bar -->
        <div class="w-full bg-carbon-surface rounded-full h-2 overflow-hidden flex border border-carbon-border">
          <div
            class="transition-all duration-500"
            :class="getQuotaPct(remote.about.used, remote.about.total) > 90 ? 'bg-rose-500' : 'bg-electric-blue'"
            :style="`width: ${getQuotaPct(remote.about.used, remote.about.total)}%`"
          ></div>
        </div>

        <!-- Quota Breakdown Metrics -->
        <div class="grid grid-cols-3 gap-2 pt-1 text-[10px] font-mono border-t border-carbon-border/50">
          <div>
            <span class="text-slate-500 block uppercase">Used Space</span>
            <span class="text-slate-200">{{ formatBytes(remote.about.used) }}</span>
          </div>
          <div>
            <span class="text-slate-500 block uppercase">Free Space</span>
            <span class="text-emerald-400">{{ formatBytes(remote.about.free) }}</span>
          </div>
          <div>
            <span class="text-slate-500 block uppercase">Trash Bin</span>
            <span class="text-amber-400">{{ formatBytes(remote.about.trashed) }}</span>
          </div>
        </div>
      </div>

      <!-- Virtual Overlays Hierarchy Section -->
      <div
        v-if="directChildren.length > 0"
        class="pt-3 border-t border-carbon-border/70 space-y-2"
      >
        <div class="flex items-center justify-between text-[11px] font-mono text-slate-400 px-0.5">
          <span class="flex items-center gap-1.5 font-bold uppercase tracking-wider text-slate-300 text-[10px]">
            <i class="pi pi-sitemap text-[11px] text-electric-blue"></i>
            <span>Virtual Overlays ({{ directChildren.length }})</span>
          </span>
          <span class="text-[10px] text-slate-500">outermost wraps innermost</span>
        </div>

        <RemoteOverlayTree
          :items="directChildren"
          :children-by-parent="childrenByParent"
          :parent-name="remote.name"
          :active-remote-name="activeRemoteName"
          @delete="emit('delete', $event)"
        />
      </div>
    </div>

    <div class="pt-3 border-t border-carbon-border flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
      <div class="flex flex-wrap items-center gap-1.5">
        <!-- Test Connection Button -->
        <button
          type="button"
          @click="remotesStore.validateRemote(remote.name)"
          :disabled="remotesStore.validating[remote.name]"
          class="px-2.5 py-1 text-[11px] rounded-lg border transition-colors flex items-center gap-1 cursor-pointer"
          :class="
            remotesStore.validated[remote.name]
              ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
              : 'bg-carbon-elevated hover:bg-slate-800 text-slate-300 border-carbon-border'
          "
          title="Test connection and remote connectivity"
        >
          <i
            :class="
              remotesStore.validating[remote.name]
                ? 'pi pi-spin pi-spinner text-electric-blue'
                : remotesStore.validated[remote.name]
                ? 'pi pi-check text-emerald-400'
                : 'pi pi-bolt text-slate-400'
            "
            class="text-xs"
          ></i>
          <span>{{ remotesStore.validated[remote.name] ? 'Connected ✓' : 'Test' }}</span>
        </button>

        <!-- OAuth Reconnect / Cancel -->
        <template v-if="remote.auth && remote.auth.is_oauth">
          <button
            v-if="!remotesStore.reconnecting[remote.name]"
            type="button"
            @click="remotesStore.reconnectRemote(remote.name)"
            class="px-2.5 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border transition-colors flex items-center gap-1 cursor-pointer"
            title="Trigger OAuth Reconnect"
          >
            <i class="pi pi-sync text-xs text-electric-blue"></i>
            <span>Reconnect</span>
          </button>
          <button
            v-else
            type="button"
            @click="remotesStore.cancelReconnect(remote.name)"
            class="px-2.5 py-1 text-[11px] rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1 cursor-pointer"
          >
            <i class="pi pi-spin pi-spinner text-xs"></i>
            <span>Cancel</span>
          </button>

          <!-- OAuth App Config -->
          <button
            type="button"
            @click="remotesStore.openMigrationWizard(remote)"
            class="px-2.5 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-slate-400 hover:text-slate-200 border border-carbon-border transition-colors flex items-center gap-1 cursor-pointer"
            title="Configure OAuth Client ID & Secret"
          >
            <i class="pi pi-key text-[10px]"></i>
            <span>OAuth</span>
          </button>
        </template>

        <!-- Hasher Remote Maintenance (if remote itself is a hasher) -->
        <button
          v-if="(remote.type || '').toLowerCase() === 'hasher'"
          type="button"
          @click="remotesStore.openHasherManager(remote.name, 'manager')"
          class="px-2.5 py-1 text-[11px] rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1 cursor-pointer font-mono"
          title="Hasher Maintenance Console"
        >
          <i class="pi pi-bolt text-xs text-amber-400"></i>
          <span>Hasher</span>
        </button>

        <!-- Base Remote Overlay Wrapping Controls -->
        <template v-else>
          <!-- Crypt Wrap Button (only allowed on non-crypt remotes) -->
          <template v-if="(remote.type || '').toLowerCase() !== 'crypt'">
            <button
              v-if="hasDirectCrypt"
              type="button"
              disabled
              class="px-2 py-1 text-[11px] rounded-lg bg-cyan-950/20 text-cyan-400/50 border border-cyan-800/30 flex items-center gap-1 cursor-not-allowed font-mono"
              title="Already wrapped by an encrypted overlay"
            >
              <i class="pi pi-shield text-xs text-cyan-400/50"></i>
              <span>Crypt ✓</span>
            </button>
            <button
              v-else
              type="button"
              @click="remotesStore.openCryptWrap(remote.name)"
              class="px-2 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-cyan-300 border border-cyan-500/30 hover:border-cyan-500/60 transition-colors flex items-center gap-1 cursor-pointer font-mono"
              title="Wrap remote with client-side Crypt encryption overlay"
            >
              <i class="pi pi-shield text-xs text-cyan-400"></i>
              <span>+ Crypt</span>
            </button>
          </template>

          <!-- Hasher Wrap Button -->
          <button
            v-if="hasDirectHasher"
            type="button"
            disabled
            class="px-2 py-1 text-[11px] rounded-lg bg-amber-950/20 text-amber-400/50 border border-amber-800/30 flex items-center gap-1 cursor-not-allowed font-mono"
            title="Already wrapped by a hasher acceleration overlay"
          >
            <i class="pi pi-bolt text-xs text-amber-400/50"></i>
            <span>Hasher ✓</span>
          </button>
          <button
            v-else
            type="button"
            @click="remotesStore.openHasherManager(remote.name, 'wrap')"
            class="px-2 py-1 text-[11px] rounded-lg bg-carbon-elevated hover:bg-slate-800 text-amber-300 border border-amber-500/30 hover:border-amber-500/60 transition-colors flex items-center gap-1 cursor-pointer font-mono"
            title="Wrap remote with Hasher checksum acceleration overlay"
          >
            <i class="pi pi-bolt text-xs text-amber-400"></i>
            <span>+ Hasher</span>
          </button>
        </template>
      </div>

      <div class="flex items-center gap-2">
        <button
          type="button"
          @click="dedupeStore.openModal(remote.name, '', 'list', remote.type)"
          class="px-2.5 py-1 rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1.5 cursor-pointer"
          title="Launch Deduplication Workspace"
        >
          <i class="pi pi-clone text-xs"></i>
          <span>Dedupe</span>
        </button>

        <button
          @click="emit('delete', remote.name)"
          class="text-rose-400 hover:text-rose-300 text-xs font-mono p-1 rounded hover:bg-rose-950/40 cursor-pointer ml-1"
          :title="directChildren.length > 0 ? `Delete remote (Warning: has ${directChildren.length} active child overlay${directChildren.length > 1 ? 's' : ''})` : 'Delete remote'"
        >
          <i class="pi pi-trash text-xs"></i>
        </button>
      </div>
    </div>
  </div>
</template>
