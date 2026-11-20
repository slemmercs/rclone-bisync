<template>
  <Dialog
    :visible="telemetryStore.showDeletionsModal"
    @update:visible="val => !val && telemetryStore.closeDeletionsModal()"
    modal
    :dismissableMask="true"
    :style="{ width: '92vw', maxWidth: '640px' }"
    :pt="{
      root: { class: 'bg-carbon-900 border border-carbon-700 rounded-2xl shadow-2xl overflow-hidden font-sans' },
      header: { class: 'hidden' },
      content: { class: 'p-0 bg-carbon-900 text-carbon-50' }
    }"
  >
    <!-- Modal Header -->
    <div class="p-5 border-b border-carbon-700/80 flex items-center justify-between bg-carbon-950/70">
      <div class="flex items-center gap-3">
        <div
          class="p-2.5 rounded-xl border shrink-0 flex items-center justify-center"
          :class="telemetryStore.hasPendingDeletions ? 'bg-rose-500/10 text-rose-400 border-rose-500/20' : 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'"
        >
          <i :class="telemetryStore.hasPendingDeletions ? 'pi pi-shield text-rose-400' : 'pi pi-check-circle text-emerald-400'" class="text-base"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white tracking-wide uppercase">Pending Deletions Review</h3>
          <div class="text-xs text-carbon-400 mt-0.5">
            {{ telemetryStore.hasPendingDeletions ? `${telemetryStore.pendingDeletions?.total_count || 0} files queued for deletion` : '0 files queued • Clean state' }}
          </div>
        </div>
      </div>
      <button
        type="button"
        @click="telemetryStore.closeDeletionsModal()"
        class="text-carbon-400 hover:text-white transition-colors p-1.5 rounded-lg hover:bg-carbon-800 cursor-pointer"
        title="Close dialog"
      >
        <i class="pi pi-times text-sm"></i>
      </button>
    </div>

    <!-- Body Section -->
    <div class="p-5 space-y-4">
      <div v-if="telemetryStore.deletionsLoading" class="py-8 text-center text-carbon-400 space-y-2">
        <i class="pi pi-spin pi-spinner text-xl text-electric-blue"></i>
        <p class="text-xs">Inspecting pending deletions ledger...</p>
      </div>

      <div v-else-if="!telemetryStore.hasPendingDeletions" class="py-8 text-center space-y-3">
        <div class="w-12 h-12 rounded-full bg-emerald-500/10 text-emerald-400 mx-auto flex items-center justify-center border border-emerald-500/20">
          <i class="pi pi-check text-xl"></i>
        </div>
        <h4 class="text-sm font-semibold text-white">No Pending Deletions</h4>
        <p class="text-xs text-carbon-400 max-w-sm mx-auto leading-relaxed">
          All prior sync operations completed safely. No deletions are currently blocked by blast-radius safety thresholds.
        </p>
      </div>

      <div v-else class="space-y-3.5">
        <div class="bg-carbon-950/80 border border-carbon-800 rounded-xl p-3.5 space-y-2 text-xs">
          <div class="flex items-center justify-between text-carbon-300">
            <span class="text-carbon-400 font-medium">Origin Source:</span>
            <span class="font-semibold text-rose-400 font-mono text-[11px]">{{ telemetryStore.pendingDeletions?.source || 'Storage Target (remote)' }}</span>
          </div>
          <div class="flex items-center justify-between text-carbon-300 border-t border-carbon-850 pt-2">
            <span class="text-carbon-400 font-medium">Target Impact:</span>
            <span class="font-semibold text-amber-400 font-mono text-[11px]">{{ telemetryStore.pendingDeletions?.target_action || 'Files queued for purge' }}</span>
          </div>
        </div>

        <div class="relative">
          <i class="pi pi-search absolute left-3 top-1/2 -translate-y-1/2 text-carbon-400 text-xs"></i>
          <input
            type="text"
            v-model="telemetryStore.deletionsSearch"
            placeholder="Filter queued files (e.g. .pdf, reports, image)..."
            class="w-full bg-carbon-950 border border-carbon-700/80 rounded-lg pl-8 pr-3 py-2 text-xs text-carbon-100 placeholder-carbon-500 focus:outline-none focus:border-electric-blue font-mono transition-colors"
          />
        </div>

        <div class="border border-carbon-700/80 rounded-xl bg-carbon-950/60 overflow-hidden">
          <div class="bg-carbon-950/90 px-3.5 py-2 border-b border-carbon-700/60 flex items-center justify-between text-[11px] font-mono text-carbon-400 uppercase tracking-wider">
            <span>Queued File Path</span>
            <span>{{ telemetryStore.filteredDeletions.length }} items</span>
          </div>
          <div class="max-h-60 overflow-y-auto p-2 space-y-1 font-mono text-xs">
            <div
              v-for="file in telemetryStore.filteredDeletions"
              :key="file"
              class="px-2.5 py-1.5 rounded bg-carbon-900/70 hover:bg-carbon-850 border border-carbon-800 text-carbon-200 flex items-center justify-between gap-2 transition-colors truncate"
            >
              <span class="truncate text-rose-300 font-mono text-[11px]" :title="file">
                <span class="text-rose-500 font-bold mr-1.5">✕</span>{{ file }}
              </span>
            </div>
            <div v-if="telemetryStore.filteredDeletions.length === 0" class="text-carbon-500 text-center py-6 font-mono text-xs">
              No queued deletion matches filter query.
            </div>
          </div>
        </div>

        <div class="p-3 bg-amber-950/20 border border-amber-600/30 rounded-xl text-[11px] text-amber-200/90 flex items-start gap-2.5">
          <i class="pi pi-exclamation-triangle text-amber-400 mt-0.5 shrink-0"></i>
          <span>These deletions exceed safety limits. Authorizing with <strong>--force</strong> will permanently purge these items during the next sync run.</span>
        </div>
      </div>
    </div>

    <!-- Modal Footer -->
    <div class="p-4 border-t border-carbon-700/80 bg-carbon-950/70 flex items-center justify-end gap-2.5">
      <button
        v-if="!telemetryStore.hasPendingDeletions"
        type="button"
        @click="telemetryStore.closeDeletionsModal()"
        class="px-4 py-2 bg-carbon-800 hover:bg-carbon-700 text-carbon-200 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
      >
        Close
      </button>
      <template v-else>
        <button
          type="button"
          @click="telemetryStore.closeDeletionsModal()"
          class="px-4 py-2 bg-carbon-800 hover:bg-carbon-700 text-carbon-300 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
        >
          Cancel & Protect Files
        </button>
        <button
          type="button"
          @click="handleAuthorizeForce"
          :disabled="authorizing"
          class="px-4 py-2 bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white rounded-lg text-xs font-bold transition-colors flex items-center justify-center gap-1.5 shadow-lg shadow-rose-950/40 cursor-pointer"
        >
          <i v-if="authorizing" class="pi pi-spin pi-spinner text-xs"></i>
          <i v-else class="pi pi-bolt text-xs"></i>
          <span>{{ authorizing ? 'Authorizing...' : 'Authorize & Delete with --force' }}</span>
        </button>
      </template>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { useTelemetryStore } from '../stores/telemetry'
import { useProfilesStore } from '../stores/profiles'
import { useToastNotification } from '../composables/useToastNotification'

const telemetryStore = useTelemetryStore()
const profilesStore = useProfilesStore()
const toast = useToastNotification()
const authorizing = ref<boolean>(false)

async function handleAuthorizeForce() {
  authorizing.value = true
  try {
    const profileId = profilesStore.selectedProfileId
    await telemetryStore.authorizeForceSync(profileId || undefined)
    toast.success('Sync authorized with --force flag', 'Deletion Override')
    telemetryStore.closeDeletionsModal()
  } catch (err: any) {
    toast.error(err?.message || 'Failed to authorize force sync', 'Authorization Error')
  } finally {
    authorizing.value = false
  }
}
</script>
