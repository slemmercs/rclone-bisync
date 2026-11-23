<script setup lang="ts">
import { useRemotesStore } from '../../stores/remotes'
import { useDedupeStore } from '../../stores/dedupe'

defineOptions({
  name: 'RemoteOverlayTree'
})

interface Props {
  items: Record<string, any>[]
  childrenByParent: Record<string, any[]>
  parentName: string
  activeRemoteName?: string | null
  depth?: number
}

const props = withDefaults(defineProps<Props>(), {
  depth: 0,
  activeRemoteName: null
})

const emit = defineEmits<{
  (e: 'delete', name: string): void
}>()

const remotesStore = useRemotesStore()
const dedupeStore = useDedupeStore()

function isActiveTarget(remoteName: string, node: Record<string, any>): boolean {
  if (node.is_active_target) return true
  if (!props.activeRemoteName) return false
  return props.activeRemoteName === remoteName
}

function hasDirectHasherChild(nodeName: string): boolean {
  const children = props.childrenByParent[nodeName] || []
  return children.some((c: any) => (c.type || '').toLowerCase() === 'hasher')
}
</script>

<template>
  <div class="flex flex-col space-y-1.5">
    <div
      v-for="(node, index) in items"
      :key="node.name"
      class="flex flex-col space-y-1.5"
    >
      <!-- Indented Overlay Node Row -->
      <div
        class="flex flex-wrap items-center justify-between gap-2 py-1.5 px-2.5 rounded-xl bg-carbon-canvas/70 border border-carbon-border/70 hover:border-carbon-border transition-colors text-xs font-mono"
        :style="{ marginLeft: `${depth * 14}px` }"
      >
        <!-- Left: Tree marker, icon, name, badge, wraps info, active -->
        <div class="flex items-center gap-2 flex-wrap min-w-0">
          <span class="text-slate-500 font-mono select-none text-xs">
            {{ index === items.length - 1 ? '└─' : '├─' }}
          </span>

          <i
            v-if="(node.type || '').toLowerCase() === 'crypt'"
            class="pi pi-shield text-cyan-400 text-xs shrink-0"
            title="Encrypted overlay"
          ></i>
          <i
            v-else-if="(node.type || '').toLowerCase() === 'hasher'"
            class="pi pi-bolt text-amber-400 text-xs shrink-0"
            title="Hasher accelerated overlay"
          ></i>
          <i v-else class="pi pi-box text-slate-400 text-xs shrink-0"></i>

          <span class="font-bold text-white text-xs tracking-tight truncate max-w-[200px]" :title="node.name">
            {{ node.name }}
          </span>

          <span
            v-if="(node.type || '').toLowerCase() === 'crypt'"
            class="text-[9px] uppercase font-bold px-1.5 py-0.5 rounded bg-cyan-950/80 text-cyan-300 border border-cyan-800/60 font-mono"
          >
            CRYPT
          </span>
          <span
            v-else-if="(node.type || '').toLowerCase() === 'hasher'"
            class="text-[9px] uppercase font-bold px-1.5 py-0.5 rounded bg-amber-950/80 text-amber-300 border border-amber-800/60 font-mono"
          >
            HASHER
          </span>
          <span
            v-else
            class="text-[9px] uppercase font-bold px-1.5 py-0.5 rounded bg-carbon-elevated text-slate-400 border border-carbon-border font-mono"
          >
            {{ (node.type || 'overlay').toUpperCase() }}
          </span>

          <span class="text-[10px] text-slate-400 font-mono">
            wraps <span class="text-cyan-400/90 font-medium">{{ node.underlying_remote || parentName }}</span>
          </span>

          <span
            v-if="isActiveTarget(node.name, node)"
            class="text-[9px] uppercase font-bold px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-800/60 font-mono"
            title="Target endpoint of active sync profile"
          >
            Active
          </span>
        </div>

        <!-- Right: Status indicator & Action controls -->
        <div class="flex items-center gap-1.5 shrink-0">
          <span
            class="text-[9px] font-mono px-2 py-0.5 rounded-full font-bold uppercase mr-1"
            :class="node.status === 'error' ? 'bg-rose-500/10 text-rose-400 border border-rose-500/30' : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'"
          >
            {{ node.status === 'error' ? 'ERR' : 'CONNECTED' }}
          </span>

          <template v-if="(node.type || '').toLowerCase() === 'crypt'">
            <button
              v-if="hasDirectHasherChild(node.name)"
              type="button"
              disabled
              class="px-2 py-0.5 text-[10px] rounded bg-amber-950/20 text-amber-400/50 border border-amber-800/30 cursor-not-allowed flex items-center gap-1 font-mono"
              title="Already accelerated with Hasher overlay"
            >
              <i class="pi pi-bolt text-[9px]"></i> Hasher ✓
            </button>
            <button
              v-else
              type="button"
              @click="remotesStore.openHasherManager(node.name, 'wrap')"
              class="px-2 py-0.5 text-[10px] rounded bg-carbon-elevated hover:bg-slate-800 text-amber-300 border border-amber-500/30 hover:border-amber-500/60 transition-colors flex items-center gap-1 cursor-pointer font-mono"
              title="Wrap with Hasher checksum acceleration overlay"
            >
              <i class="pi pi-plus text-[9px]"></i> Hasher
            </button>
          </template>

          <template v-else-if="(node.type || '').toLowerCase() === 'hasher'">
            <button
              type="button"
              @click="remotesStore.openHasherManager(node.name, 'manager')"
              class="px-2 py-0.5 text-[10px] rounded bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1 cursor-pointer font-mono"
              title="Hasher Maintenance Console"
            >
              <i class="pi pi-bolt text-[9px]"></i> Console
            </button>
          </template>

          <button
            type="button"
            @click="remotesStore.validateRemote(node.name)"
            :disabled="remotesStore.validating[node.name]"
            class="px-2 py-0.5 text-[10px] rounded border transition-colors flex items-center gap-1 cursor-pointer font-mono"
            :class="
              remotesStore.validated[node.name]
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                : 'bg-carbon-elevated hover:bg-slate-800 text-slate-300 border-carbon-border'
            "
            title="Test connectivity through this overlay"
          >
            <i
              :class="
                remotesStore.validating[node.name]
                  ? 'pi pi-spin pi-spinner text-electric-blue'
                  : remotesStore.validated[node.name]
                  ? 'pi pi-check text-emerald-400'
                  : 'pi pi-bolt text-slate-400'
              "
              class="text-[9px]"
            ></i>
            <span>{{ remotesStore.validated[node.name] ? 'OK' : 'Test' }}</span>
          </button>

          <button
            type="button"
            @click="dedupeStore.openModal(node.name, '', 'list', node.type)"
            class="px-2 py-0.5 text-[10px] rounded bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 transition-colors flex items-center gap-1 cursor-pointer font-mono"
            title="Launch Deduplication Workspace"
          >
            <i class="pi pi-clone text-[9px]"></i>
            <span>Dedupe</span>
          </button>

          <button
            type="button"
            @click="emit('delete', node.name)"
            class="text-rose-400 hover:text-rose-300 p-1 rounded hover:bg-rose-950/40 transition-colors cursor-pointer"
            title="Delete overlay"
          >
            <i class="pi pi-trash text-[10px]"></i>
          </button>
        </div>
      </div>

      <!-- Recursive Descendant Tree (e.g. Hasher on Crypt) -->
      <RemoteOverlayTree
        v-if="childrenByParent[node.name] && childrenByParent[node.name].length > 0"
        :items="childrenByParent[node.name]"
        :children-by-parent="childrenByParent"
        :parent-name="node.name"
        :active-remote-name="activeRemoteName"
        :depth="depth + 1"
        @delete="emit('delete', $event)"
      />
    </div>
  </div>
</template>
