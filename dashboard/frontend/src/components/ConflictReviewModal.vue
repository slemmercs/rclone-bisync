<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useReviewStore } from '../stores/review'
import { useToastNotification } from '../composables/useToastNotification'
import type { ActionDecision } from '../types/models'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const reviewStore = useReviewStore()
const toast = useToastNotification()
const confirmingExecution = ref<boolean>(false)

watch(
  () => props.visible,
  (val) => {
    if (val) {
      reviewStore.fetchActiveReview()
    }
  }
)

onMounted(async () => {
  await reviewStore.fetchActiveReview()
})

const session = computed(() => reviewStore.session)
const deletions = computed(() => session.value?.deletions || [])
const conflicts = computed(() => session.value?.conflicts || [])
const pendingCount = computed(() => reviewStore.pendingDeletionsCount + reviewStore.pendingConflictsCount)
const stagedCount = computed(() => {
  const cCount = conflicts.value.filter((c) => !!c.decision).length
  const dCount = deletions.value.filter((d) => !!d.decision).length
  return cCount + dCount
})

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

async function autoApplyHeuristics() {
  try {
    const staged = await reviewStore.applyBatchRecommendations(0.7, 'MEDIUM')
    toast.success(`Auto-staged recommendations for ${staged} candidate(s).`, 'Heuristics Applied')
  } catch (err: any) {
    toast.error(err.message || 'Failed to auto-apply recommendations', 'Auto-Apply Error')
  }
}

async function executeResolutions() {
  try {
    await reviewStore.applyDecisions()
    confirmingExecution.value = false
    toast.success('Review resolutions executed with pre-mutation backups.', 'Sync Finalized')
    emit('update:visible', false)
  } catch (err: any) {
    toast.error(err.message || 'Failed to execute resolutions', 'Resolution Error')
  }
}

function close() {
  confirmingExecution.value = false
  emit('update:visible', false)
}

function getRiskBadgeClass(risk?: string) {
  switch (risk) {
    case 'LOW':
      return 'bg-emerald-950/80 text-emerald-300 border border-emerald-600/60'
    case 'MEDIUM':
      return 'bg-amber-950/80 text-amber-300 border border-amber-600/60'
    case 'HIGH':
      return 'bg-rose-950/80 text-rose-300 border border-rose-600/60'
    case 'CRITICAL':
      return 'bg-purple-950/80 text-purple-300 border border-purple-600/60'
    default:
      return 'bg-carbon-800 text-carbon-300 border border-carbon-700'
  }
}

function getRiskTierClass(risk?: string) {
  switch (risk) {
    case 'LOW':
      return 'text-emerald-400 border-emerald-700/50 bg-emerald-950/30'
    case 'MEDIUM':
      return 'text-amber-400 border-amber-700/50 bg-amber-950/30'
    case 'HIGH':
      return 'text-rose-400 border-rose-700/50 bg-rose-950/30'
    case 'CRITICAL':
      return 'text-purple-400 border-purple-700/50 bg-purple-950/30'
    default:
      return 'text-carbon-400 border-carbon-700 bg-carbon-900'
  }
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
            <span v-if="session?.profile_id" class="px-2 py-0.5 rounded bg-carbon-800 text-electric-400 font-mono text-[10px]">
              Profile: {{ session.profile_id }}
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
          @click="autoApplyHeuristics"
          :disabled="reviewStore.loading"
          class="px-2.5 py-1 rounded bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border border-amber-500/50 font-semibold transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-40"
          title="Auto-apply rule-based heuristic recommendations (minimum 70% confidence, medium risk ceiling)"
        >
          <i class="pi pi-bolt text-xs text-amber-400"></i>
          <span>Auto-Apply Recommendations</span>
        </button>
        <button
          v-if="conflicts.length > 0"
          @click="batchDecideConflicts('keep-local')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-electric-300 border border-carbon-700 transition-colors cursor-pointer"
        >
          Keep All Local
        </button>
        <button
          v-if="conflicts.length > 0"
          @click="batchDecideConflicts('keep-remote')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-electric-300 border border-carbon-700 transition-colors cursor-pointer"
        >
          Keep All Remote
        </button>
        <button
          v-if="conflicts.length > 0"
          @click="batchDecideConflicts('keep-both')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-indigo-300 border border-carbon-700 transition-colors cursor-pointer"
        >
          Keep Both (Rename)
        </button>
        <button
          v-if="deletions.length > 0"
          @click="batchDecideDeletions('restore-item')"
          class="px-2 py-1 rounded bg-carbon-800 hover:bg-carbon-750 text-emerald-300 border border-carbon-700 transition-colors cursor-pointer"
        >
          Restore All Deletions
        </button>
      </div>
      <!-- Candidate Items List -->
      <div class="max-h-[380px] overflow-y-auto space-y-2 pr-1">
        <!-- Conflict Candidates -->
        <div
          v-for="c in conflicts"
          :key="c.id"
          class="p-3 rounded-lg bg-carbon-950 border border-carbon-800 flex flex-col gap-2 transition-colors"
        >
          <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
            <div class="space-y-1 min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="px-1.5 py-0.2 rounded font-mono text-[10px] font-bold uppercase bg-amber-950 text-amber-400 border border-amber-800">
                  conflict
                </span>
                <span class="font-mono text-carbon-200 text-xs truncate" :title="c.path">{{ c.name || c.path }}</span>

                <!-- Heuristic Badge -->
                <div v-if="c.recommendation" class="flex items-center gap-1.5 flex-wrap">
                  <span
                    class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold uppercase flex items-center gap-1"
                    :class="getRiskBadgeClass(c.recommendation.risk || c.recommendation.risk_tier)"
                  >
                    <i class="pi pi-bolt text-[9px]"></i>
                    {{ c.recommendation.suggested_action }} ({{ Math.round(c.recommendation.confidence * 100) }}%)
                  </span>
                  <span
                    class="px-1.5 py-0.5 rounded text-[9px] font-mono border"
                    :class="getRiskTierClass(c.recommendation.risk || c.recommendation.risk_tier)"
                  >
                    {{ c.recommendation.risk || c.recommendation.risk_tier || 'LOW' }} RISK
                  </span>
                </div>
              </div>

              <div class="text-[11px] text-carbon-500 font-mono flex items-center gap-3 flex-wrap">
                <span>Local: {{ c.local_size }} B</span>
                <span>Remote: {{ c.remote_size }} B</span>
                <span v-if="c.decision" class="text-electric-400 font-semibold">
                  Staged: {{ c.decision }}
                </span>
              </div>

              <!-- Recommendation Reason -->
              <div v-if="c.recommendation?.primary_reason" class="text-[11px] text-amber-300/80 italic font-sans flex items-center gap-1.5 pt-0.5">
                <i class="pi pi-info-circle text-[10px] text-amber-400 shrink-0"></i>
                <span>{{ c.recommendation.primary_reason }}</span>
              </div>
            </div>

            <!-- Resolution Decision Buttons -->
            <div class="flex items-center gap-1.5 shrink-0 font-mono text-[11px]">
              <button
                @click="stageCandidateDecision('conflict', c.id, 'keep-local')"
                class="px-2 py-1 rounded border transition-colors cursor-pointer"
                :class="c.decision === 'keep-local' ? 'bg-electric-500 text-white border-electric-400 font-bold' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
              >
                Keep Local
              </button>
              <button
                @click="stageCandidateDecision('conflict', c.id, 'keep-remote')"
                class="px-2 py-1 rounded border transition-colors cursor-pointer"
                :class="c.decision === 'keep-remote' ? 'bg-electric-500 text-white border-electric-400 font-bold' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
              >
                Keep Remote
              </button>
              <button
                @click="stageCandidateDecision('conflict', c.id, 'keep-both')"
                class="px-2 py-1 rounded border transition-colors cursor-pointer"
                :class="c.decision === 'keep-both' ? 'bg-indigo-600 text-white border-indigo-500 font-bold' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
              >
                Keep Both
              </button>
            </div>
          </div>
        </div>

        <!-- Deletion Candidates -->
        <div
          v-for="d in deletions"
          :key="d.id"
          class="p-3 rounded-lg bg-carbon-950 border border-carbon-800 flex flex-col gap-2 transition-colors"
        >
          <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
            <div class="space-y-1 min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="px-1.5 py-0.2 rounded font-mono text-[10px] font-bold uppercase bg-red-950 text-red-400 border border-red-800">
                  deletion
                </span>
                <span class="font-mono text-carbon-200 text-xs truncate" :title="d.path">{{ d.name || d.path }}</span>

                <!-- Heuristic Badge if available -->
                <div v-if="d.recommendation" class="flex items-center gap-1.5 flex-wrap">
                  <span
                    class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold uppercase flex items-center gap-1"
                    :class="getRiskBadgeClass(d.recommendation.risk || d.recommendation.risk_tier)"
                  >
                    <i class="pi pi-bolt text-[9px]"></i>
                    {{ d.recommendation.suggested_action }} ({{ Math.round(d.recommendation.confidence * 100) }}%)
                  </span>
                  <span
                    class="px-1.5 py-0.5 rounded text-[9px] font-mono border"
                    :class="getRiskTierClass(d.recommendation.risk || d.recommendation.risk_tier)"
                  >
                    {{ d.recommendation.risk || d.recommendation.risk_tier || 'LOW' }} RISK
                  </span>
                </div>
              </div>

              <div class="text-[11px] text-carbon-500 font-mono flex items-center gap-3 flex-wrap">
                <span>Source: {{ d.source }}</span>
                <span>Size: {{ d.size ? `${d.size} B` : '--' }}</span>
                <span v-if="d.decision" class="text-electric-400 font-semibold">
                  Staged: {{ d.decision }}
                </span>
              </div>

              <!-- Recommendation Reason -->
              <div v-if="d.recommendation?.primary_reason" class="text-[11px] text-amber-300/80 italic font-sans flex items-center gap-1.5 pt-0.5">
                <i class="pi pi-info-circle text-[10px] text-amber-400 shrink-0"></i>
                <span>{{ d.recommendation.primary_reason }}</span>
              </div>
            </div>

            <div class="flex items-center gap-1.5 shrink-0 font-mono text-[11px]">
              <button
                @click="stageCandidateDecision('deletion', d.id, 'accept-delete')"
                class="px-2 py-1 rounded border transition-colors cursor-pointer"
                :class="d.decision === 'accept-delete' ? 'bg-red-600 text-white border-red-500 font-bold' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
              >
                Confirm Delete
              </button>
              <button
                @click="stageCandidateDecision('deletion', d.id, 'restore-item')"
                class="px-2 py-1 rounded border transition-colors cursor-pointer"
                :class="d.decision === 'restore-item' ? 'bg-emerald-600 text-white border-emerald-500 font-bold' : 'bg-carbon-850 text-carbon-300 border-carbon-700 hover:text-white'"
              >
                Restore Item
              </button>
            </div>
          </div>
        </div>

        <div v-if="conflicts.length === 0 && deletions.length === 0" class="p-8 text-center text-carbon-500">
          No review candidates awaiting decisions.
        </div>
      </div>

      <!-- Footer & Execution -->
      <div class="border-t border-carbon-800 pt-3 flex flex-wrap items-center justify-between gap-3">
        <button
          @click="close"
          class="px-3 py-1.5 rounded bg-carbon-850 hover:bg-carbon-800 text-carbon-300 text-xs font-semibold cursor-pointer"
        >
          Dismiss
        </button>

        <div class="flex items-center gap-2">
          <template v-if="!confirmingExecution">
            <button
              @click="confirmingExecution = true"
              :disabled="(conflicts.length === 0 && deletions.length === 0) || reviewStore.applying"
              class="px-4 py-1.5 rounded bg-electric-500 hover:bg-electric-400 text-white text-xs font-semibold transition-all shadow-sm disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer flex items-center gap-1.5"
            >
              <i class="pi pi-check text-xs"></i>
              <span>Apply Resolutions ({{ stagedCount }} Staged)</span>
            </button>
          </template>

          <template v-else>
            <span class="text-amber-300 text-[11px] font-medium mr-1">
              Apply {{ stagedCount }} staged resolution(s) with pre-mutation backups?
            </span>
            <button
              @click="confirmingExecution = false"
              class="px-3 py-1.5 rounded bg-carbon-800 hover:bg-carbon-750 text-carbon-300 text-xs font-medium cursor-pointer"
            >
              Cancel
            </button>
            <button
              @click="executeResolutions"
              :disabled="reviewStore.applying"
              class="px-4 py-1.5 rounded bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition-all shadow-sm cursor-pointer flex items-center gap-1.5"
            >
              <i v-if="reviewStore.applying" class="pi pi-spin pi-spinner text-xs"></i>
              <span>Confirm & Apply</span>
            </button>
          </template>
        </div>
      </div>
    </div>
  </Dialog>
</template>
