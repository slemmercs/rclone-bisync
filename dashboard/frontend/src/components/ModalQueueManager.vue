<template>
  <Dialog
    :visible="queueStore.isModalOpen"
    @update:visible="val => !val && queueStore.closeModal()"
    modal
    header="Sync Queue & Concurrency Dispatcher"
    :style="{ width: '92vw', maxWidth: '880px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border shadow-2xl rounded-2xl overflow-hidden' },
      header: { class: 'p-4 border-b border-carbon-border bg-carbon-surface text-white flex items-center justify-between' },
      content: { class: 'p-5 bg-carbon-canvas text-xs font-mono space-y-5 max-h-[75vh] overflow-y-auto' },
      footer: { class: 'p-3 border-t border-carbon-border bg-carbon-surface flex justify-between items-center' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div
          class="w-8 h-8 rounded-xl border flex items-center justify-center shrink-0"
          :class="queueStore.isPaused
            ? 'bg-amber-500/10 text-amber-400 border-amber-500/30'
            : (queueStore.runningCount > 0 ? 'bg-electric-500/10 text-electric-400 border-electric-500/30' : 'bg-carbon-800 text-carbon-400 border-carbon-700')"
        >
          <i class="pi text-base" :class="queueStore.runningCount > 0 ? 'pi-sync pi-spin' : 'pi-list'"></i>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-sm font-bold text-white tracking-wide">Sync Queue & Concurrency Dispatcher</h3>
            <span
              class="text-[10px] uppercase font-mono px-2 py-0.5 rounded border"
              :class="queueStore.isPaused
                ? 'bg-amber-950 text-amber-300 border-amber-800/60'
                : 'bg-emerald-950 text-emerald-300 border-emerald-800/60'"
            >
              {{ queueStore.isPaused ? 'DISPATCHER PAUSED' : 'ACTIVE' }}
            </span>
          </div>
          <p class="text-[11px] text-slate-400">
            Active Cohort Signature:
            <span class="font-bold text-electric-blue">{{ queueStore.activeSignature || 'SIG_DEFAULT' }}</span>
            <span class="mx-1.5 text-slate-600">•</span>
            Max Concurrency:
            <span class="text-white font-bold">{{ queueStore.snapshot?.max_concurrency ?? 2 }}</span>
          </p>
        </div>
      </div>
    </template>

    <!-- Queue Action Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3 p-3 bg-carbon-surface border border-carbon-border rounded-xl">
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5 text-xs text-slate-300">
          <span class="w-2 h-2 rounded-full" :class="queueStore.runningCount > 0 ? 'bg-emerald-400 animate-pulse' : 'bg-slate-600'"></span>
          <span>Running: <strong class="text-white">{{ queueStore.runningCount }}</strong></span>
        </div>
        <span class="text-slate-600">•</span>
        <div class="flex items-center gap-1.5 text-xs text-slate-300">
          <span class="w-2 h-2 rounded-full" :class="queueStore.queuedCount > 0 ? 'bg-electric-400' : 'bg-slate-600'"></span>
          <span>Queued: <strong class="text-white">{{ queueStore.queuedCount }}</strong></span>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <button
          v-if="!queueStore.isPaused"
          type="button"
          @click="handlePause"
          :disabled="queueStore.actionLoading"
          class="px-2.5 py-1.2 rounded-lg bg-amber-950/70 hover:bg-amber-900 border border-amber-700/80 text-amber-300 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
          title="Pause new sync job dispatches"
        >
          <i class="pi pi-pause text-xs"></i>
          <span>Pause</span>
        </button>
        <button
          v-else
          type="button"
          @click="handleResume"
          :disabled="queueStore.actionLoading"
          class="px-2.5 py-1.2 rounded-lg bg-emerald-950/70 hover:bg-emerald-900 border border-emerald-700/80 text-emerald-300 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
          title="Resume dispatching queued jobs"
        >
          <i class="pi pi-play text-xs"></i>
          <span>Resume</span>
        </button>

        <button
          type="button"
          @click="handleClear"
          :disabled="queueStore.actionLoading || queueStore.queuedCount === 0"
          class="px-2.5 py-1.2 rounded-lg bg-rose-950/60 hover:bg-rose-900/80 border border-rose-800/80 text-rose-300 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
          title="Cancel and remove all queued sync jobs"
        >
          <i class="pi pi-trash text-xs"></i>
          <span>Clear</span>
        </button>

        <button
          type="button"
          @click="queueStore.fetchQueue()"
          class="px-2.5 py-1.2 rounded-lg bg-carbon-800 hover:bg-carbon-700 border border-carbon-border text-slate-300 hover:text-white text-xs flex items-center gap-1.5 transition-colors cursor-pointer"
          title="Refresh queue state"
        >
          <i class="pi pi-refresh text-xs"></i>
        </button>
      </div>
    </div>

    <!-- Error Banner -->
    <div v-if="queueStore.error" class="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs flex items-center justify-between">
      <div class="flex items-center gap-2">
        <i class="pi pi-exclamation-circle text-rose-400"></i>
        <span>{{ queueStore.error }}</span>
      </div>
      <button @click="queueStore.error = null" class="text-rose-400 hover:text-rose-200">
        <i class="pi pi-times text-xs"></i>
      </button>
    </div>

    <!-- Active Executions Section -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <h4 class="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center gap-2">
          <i class="pi pi-spin pi-spinner text-electric-400 text-xs"></i>
          <span>Active Executions ({{ queueStore.runningCount }})</span>
        </h4>
        <span class="text-[11px] text-slate-500 font-mono">
          Matching Signature: {{ queueStore.activeSignature || 'None' }}
        </span>
      </div>

      <div v-if="queueStore.activeJobs.length === 0" class="p-4 bg-carbon-surface border border-carbon-border rounded-xl text-center text-slate-400">
        <i class="pi pi-check-circle text-emerald-400 text-base mb-1 block"></i>
        <span>No active sync jobs currently running. Dispatcher is idle.</span>
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="job in queueStore.activeJobs"
          :key="job.profile_id"
          class="p-3 bg-carbon-surface border border-carbon-border rounded-xl flex flex-wrap items-center justify-between gap-3"
        >
          <div class="space-y-1">
            <div class="flex items-center gap-2">
              <span class="text-xs font-bold text-white">{{ job.profile_name || job.profile_id }}</span>
              <span class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-carbon-800 text-slate-400 border border-carbon-700">
                {{ job.profile_id }}
              </span>
              <span v-if="job.dry_run" class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-amber-950/80 text-amber-300 border border-amber-800/80">DRY-RUN</span>
              <span v-if="job.resync" class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-purple-950/80 text-purple-300 border border-purple-800/80">RESYNC</span>
              <span v-if="job.force" class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-rose-950/80 text-rose-300 border border-rose-800/80">FORCE</span>
            </div>
            <div class="flex items-center gap-3 text-[11px] text-slate-400">
              <span>Origin: <strong class="text-slate-300">{{ job.origin }}</strong></span>
              <span>•</span>
              <span>Signature: <strong class="text-electric-blue">{{ job.signature }}</strong></span>
              <span v-if="job.started_at">•</span>
              <span v-if="job.started_at">Started: {{ formatTime(job.started_at) }}</span>
            </div>
          </div>
          <span class="text-xs font-mono text-emerald-400 flex items-center gap-1.5 px-2 py-1 rounded bg-emerald-950/40 border border-emerald-800/60">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping"></span>
            Running
          </span>
        </div>
      </div>
    </div>

    <!-- Pending Queue Section -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <h4 class="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center gap-2">
          <i class="pi pi-clock text-amber-400 text-xs"></i>
          <span>Pending Queue ({{ queueStore.queuedCount }})</span>
        </h4>
        <span class="text-[11px] text-slate-500 font-mono">Fair FIFO with Aging SLA</span>
      </div>

      <div v-if="queueStore.queuedJobs.length === 0" class="p-4 bg-carbon-surface border border-carbon-border rounded-xl text-center text-slate-400">
        <i class="pi pi-inbox text-slate-500 text-base mb-1 block"></i>
        <span>Queue is empty. Incoming jobs will be executed without delay.</span>
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="(job, index) in queueStore.queuedJobs"
          :key="job.profile_id"
          class="p-3 bg-carbon-surface border border-carbon-border rounded-xl flex flex-wrap items-center justify-between gap-3"
        >
          <div class="flex items-center gap-3">
            <span class="w-6 h-6 rounded-lg bg-carbon-800 border border-carbon-700 flex items-center justify-center text-xs font-bold text-slate-300">
              #{{ index + 1 }}
            </span>
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <span class="text-xs font-bold text-white">{{ job.profile_name || job.profile_id }}</span>
                <span class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-carbon-800 text-slate-400 border border-carbon-700">
                  {{ job.profile_id }}
                </span>
                <span v-if="job.dry_run" class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-amber-950/80 text-amber-300 border border-amber-800/80">DRY-RUN</span>
                <span v-if="job.resync" class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-purple-950/80 text-purple-300 border border-purple-800/80">RESYNC</span>
              </div>
              <div class="flex items-center gap-3 text-[11px] text-slate-400">
                <span>Origin: <strong class="text-slate-300">{{ job.origin }}</strong></span>
                <span>•</span>
                <span>Wait: <strong class="text-amber-300 font-mono">{{ formatWait(job.wait_seconds) }}</strong></span>
                <span>•</span>
                <span>
                  Signature:
                  <span
                    class="font-mono text-[10px] px-1 py-0.2 rounded"
                    :class="job.signature === queueStore.activeSignature
                      ? 'bg-electric-950 text-electric-300 border border-electric-700/60'
                      : 'bg-carbon-800 text-slate-400 border border-carbon-700'"
                  >
                    {{ job.signature }}
                    <span v-if="job.signature === queueStore.activeSignature" class="text-[9px] text-electric-400"> (matching)</span>
                  </span>
                </span>
              </div>
            </div>
          </div>

          <button
            type="button"
            @click="handleCancel(job.profile_id)"
            :disabled="cancellingId === job.profile_id"
            class="px-2.5 py-1 rounded-lg bg-rose-950/50 hover:bg-rose-900 border border-rose-800/70 text-rose-300 text-xs font-mono transition-colors cursor-pointer flex items-center gap-1.5 disabled:opacity-50"
            title="Remove this job from the queue"
          >
            <i class="pi" :class="cancellingId === job.profile_id ? 'pi-spin pi-spinner' : 'pi-times'"></i>
            <span>Cancel</span>
          </button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center gap-2 text-[11px] text-slate-400 font-mono">
        <i class="pi pi-info-circle text-electric-blue"></i>
        <span>Dynamic batching switches Rclone global flags (`/options/set`) between signature cohorts.</span>
      </div>

      <button
        type="button"
        @click="queueStore.closeModal()"
        class="px-4 py-1.5 rounded-xl border border-carbon-border bg-carbon-surface hover:bg-carbon-800 text-slate-300 hover:text-white cursor-pointer transition-colors"
      >
        Close
      </button>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import { useQueueStore } from '../stores/queue'

const queueStore = useQueueStore()
const cancellingId = ref<string | null>(null)

async function handlePause() {
  try {
    await queueStore.pauseQueue()
  } catch (_) {
    // Handled in store error
  }
}

async function handleResume() {
  try {
    await queueStore.resumeQueue()
  } catch (_) {
    // Handled in store error
  }
}

async function handleClear() {
  if (!confirm('Are you sure you want to cancel and remove all pending jobs in the queue?')) {
    return
  }
  try {
    await queueStore.clearQueue()
  } catch (_) {
    // Handled in store error
  }
}

async function handleCancel(profileId: string) {
  cancellingId.value = profileId
  try {
    await queueStore.cancelJob(profileId)
  } catch (_) {
    // Handled in store error
  } finally {
    cancellingId.value = null
  }
}

function formatWait(seconds: number): string {
  if (seconds < 60) {
    return `${Math.round(seconds)}s`
  }
  const mins = Math.floor(seconds / 60)
  const remSec = Math.round(seconds % 60)
  return `${mins}m ${remSec}s`
}

function formatTime(isoStr: string): string {
  try {
    const d = new Date(isoStr)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch (_) {
    return isoStr
  }
}
</script>
