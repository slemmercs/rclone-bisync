<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import { useReviewStore } from '../stores/review'
import type { ActionDecision } from '../types/models'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const reviewStore = useReviewStore()
const confirmationToken = ref<string>('')
const confirmingExecution = ref<boolean>(false)

onMounted(async () => {
  await reviewStore.fetchActiveReview()
})

const session = computed(() => reviewStore.session)
const deletions = computed(() => session.value?.deletions || [])
const conflicts = computed(() => session.value?.conflicts || [])
const pendingCount = computed(() => reviewStore.pendingDeletionsCount + reviewStore.pendingConflictsCount)

async function stageCandidateDecision(type: 'deletion' | 'conflict', id: string, decision: ActionDecision) {
  await reviewStore.stageDecision(type, id, decision)
}

async function batchDecideConflicts(decision: ActionDecision) {
  for (const c of conflicts.value) {
    await reviewStore.stageDecision('conflict', c.id, decision)
  }
}

async function batchDecideDeletions(decision: ActionDecision) {
  for (const d of deletions.value) {
    await reviewStore.stageDecision('deletion', d.id, decision)
  }
}

async function executeResolutions() {
  if (!confirmationToken.value) return
  await reviewStore.applyDecisions(confirmationToken.value)
  confirmingExecution.value = false
  confirmationToken.value = ''
  emit('update:visible', false)
}

function close() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="props.visible"
    @update:visible="(val) => emit('update:visible', val)"
    modal
    header="Sync Conflict Review & Circuit Breaker Resolution"
    :style="{ width: '56rem', maxWidth: '95vw' }"
    :closable="true"
    class="bg-carbon-900 border border-carbon-700 text-carbon-50 shadow-2xl"
  >
    <div class="space-y-4 text-xs font-sans">
      <!-- Session Subheader -->
      <div class="bg-carbon-950 p-3 rounded-lg border border-carbon-800 flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="flex items-center gap-2">
            <span class="font-bold text-carbon-100 uppercase tracking-wide">Review. Decide. Sync.</span>
            <span class="px-2 py-0.5 rounded bg-carbon-800 text-carbon-400 font-mono text-[10px]">
              Session: {{ session?.id || 'NO ACTIVE SESSION' }}
            </span>
          </div>
          <p class="text-[11px] text-carbon-400 mt-0.5">
            Stage per-item resolution decisions. All mutations create pre-mutation safety backups before execution.
          </p>
        </div>

        <!-- Quick Summary Counters -->
        <div class="flex items-center gap-2 font-mono text-[11px]">
          <span class="px-2 py-1 rounded bg-amber-950/80 border border-amber-600/60 text-amber-300">
            {{ conflicts.length }} Conflicts
          </span>
          <span class="px-2 py-1 rounded bg-red-950/80 border border-red-600/60 text-red-300">
            {{ deletions.length }} Deletions
          </span>
        </div>
      </div>

      <!-- Batch Actions Bar -->
      <div v-if="conflicts.length > 0 || deletions.length > 0" class="flex flex-wrap items-center gap-2 p-2 bg-carbon-850 rounded border border-carbon-800 text-[11px]">
        <span class="text-carbon-400 font-semibold mr-1">Batch Actions:</span>
        <button
          v-if="conflicts.length > 0"
          @click="batchDecideConflicts('keep-local')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-electric-300 border border-carbon-700 transition-colors"
        >
          Keep All Local
        </button>
        <button
          v-if="conflicts.length > 0"
          @click="batchDecideConflicts('keep-remote')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-electric-300 border border-carbon-700 transition-colors"
        >
          Keep All Remote
        </button>
        <button
          v-if="conflicts.length > 0"
          @click="batchDecideConflicts('keep-both')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-indigo-300 border border-carbon-700 transition-colors"
        >
          Keep Both (Rename)
        </button>
        <button
          v-if="deletions.length > 0"
          @click="batchDecideDeletions('restore-item')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-emerald-300 border border-carbon-700 transition-colors"
        >
          Restore All Deletions
        </button>
      </div>
      <!-- Candidate Items List -->
      <div class="max-h-[360px] overflow-y-auto space-y-2 pr-1">
        <!-- Conflict Candidates -->
        <div
          v-for="c in conflicts"
          :key="c.id"
          class="p-3 rounded-lg bg-carbon-950 border border-carbon-800 flex flex-col md:flex-row md:items-center justify-between gap-3"
        >
          <div class="space-y-1 min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="px-1.5 py-0.2 rounded font-mono text-[10px] font-bold uppercase bg-amber-950 text-amber-400 border border-amber-800">
                conflict
              </span>
              <span class="font-mono text-carbon-200 text-xs truncate" :title="c.path">{{ c.name || c.path }}</span>
            </div>
            <div class="text-[11px] text-carbon-500 font-mono flex items-center gap-3">
              <span>Local: {{ c.local_size }} B</span>
              <span>Remote: {{ c.remote_size }} B</span>
              <span v-if="c.decision" class="text-electric-400 font-semibold">
                Staged: {{ c.decision }}
              </span>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0 font-mono text-[11px]">
            <button
              @click="stageCandidateDecision('conflict', c.id, 'keep-local')"
              class="px-2 py-1 rounded border transition-colors"
              :class="c.decision === 'keep-local' ? 'bg-electric-500 text-white border-electric-400' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
            >
              Keep Local
            </button>
            <button
              @click="stageCandidateDecision('conflict', c.id, 'keep-remote')"
              class="px-2 py-1 rounded border transition-colors"
              :class="c.decision === 'keep-remote' ? 'bg-electric-500 text-white border-electric-400' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
            >
              Keep Remote
            </button>
            <button
              @click="stageCandidateDecision('conflict', c.id, 'keep-both')"
              class="px-2 py-1 rounded border transition-colors"
              :class="c.decision === 'keep-both' ? 'bg-indigo-600 text-white border-indigo-500' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
            >
              Keep Both
            </button>
          </div>
        </div>

        <!-- Deletion Candidates -->
        <div
          v-for="d in deletions"
          :key="d.id"
          class="p-3 rounded-lg bg-carbon-950 border border-carbon-800 flex flex-col md:flex-row md:items-center justify-between gap-3"
        >
          <div class="space-y-1 min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="px-1.5 py-0.2 rounded font-mono text-[10px] font-bold uppercase bg-red-950 text-red-400 border border-red-800">
                deletion
              </span>
              <span class="font-mono text-carbon-200 text-xs truncate" :title="d.path">{{ d.name || d.path }}</span>
            </div>
            <div class="text-[11px] text-carbon-500 font-mono flex items-center gap-3">
              <span>Source: {{ d.source }}</span>
              <span v-if="d.decision" class="text-electric-400 font-semibold">
                Staged: {{ d.decision }}
              </span>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0 font-mono text-[11px]">
            <button
              @click="stageCandidateDecision('deletion', d.id, 'accept-delete')"
              class="px-2 py-1 rounded border transition-colors"
              :class="d.decision === 'accept-delete' ? 'bg-red-600 text-white border-red-500' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
            >
              Confirm Delete
            </button>
            <button
              @click="stageCandidateDecision('deletion', d.id, 'restore-item')"
              class="px-2 py-1 rounded border transition-colors"
              :class="d.decision === 'restore-item' ? 'bg-emerald-600 text-white border-emerald-500' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
            >
              Restore Item
            </button>
          </div>
        </div>

        <div v-if="conflicts.length === 0 && deletions.length === 0" class="p-8 text-center text-carbon-500">
          No review candidates awaiting decisions.
        </div>
      </div>

      <!-- Footer & Confirmation Execution -->
      <div class="border-t border-carbon-800 pt-3 flex flex-wrap items-center justify-between gap-3">
        <button
          @click="close"
          class="px-3 py-1.5 rounded bg-carbon-850 hover:bg-carbon-800 text-carbon-300 text-xs font-semibold"
        >
          Dismiss
        </button>

        <div class="flex items-center gap-2">
          <input
            v-if="confirmingExecution"
            v-model="confirmationToken"
            type="text"
            placeholder="Confirmation token..."
            class="bg-carbon-950 border border-carbon-700 text-xs font-mono text-carbon-200 px-2.5 py-1.5 rounded focus:outline-none focus:border-electric-500"
          />

          <button
            v-if="!confirmingExecution"
            @click="confirmingExecution = true"
            :disabled="conflicts.length === 0 && deletions.length === 0"
            class="px-4 py-1.5 rounded bg-electric-500 hover:bg-electric-400 text-white text-xs font-semibold transition-all shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Authorize Decisions ({{ pendingCount }} Pending)
          </button>

          <button
            v-else
            @click="executeResolutions"
            :disabled="!confirmationToken"
            class="px-4 py-1.5 rounded bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition-all shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Confirm & Apply
          </button>
        </div>
      </div>
    </div>
  </Dialog>
</template>
