<template>
  <Drawer
    :visible="helpStore.isOpen"
    @update:visible="val => !val && helpStore.closeDrawer()"
    position="right"
    :style="{ width: '95vw', maxWidth: '680px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border-l border-carbon-border shadow-2xl overflow-hidden' },
      header: { class: 'p-4 border-b border-carbon-border bg-carbon-surface flex justify-between items-center text-white' },
      content: { class: 'p-0 bg-carbon-canvas flex flex-col h-full overflow-hidden' },
      closeButton: { class: 'text-slate-400 hover:text-white hover:bg-carbon-elevated rounded-lg p-1.5 transition-colors cursor-pointer' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5 min-w-0">
        <div class="w-8 h-8 rounded-xl bg-electric-blue/10 border border-electric-blue/30 text-electric-blue flex items-center justify-center shrink-0">
          <i class="pi pi-question-circle text-base"></i>
        </div>
        <div class="min-w-0">
          <h2 class="text-sm font-bold text-white tracking-wide truncate">Reference Manual & Diagnostics</h2>
          <p class="text-[11px] text-slate-400 truncate">Policy guide, flag reference, runbooks, and live audits</p>
        </div>
      </div>
    </template>

    <!-- Drawer Toolbar -->
    <div class="shrink-0 border-b border-carbon-border p-4 bg-carbon-surface/60 flex flex-col gap-3">
      <!-- Search Bar -->
      <div class="relative">
        <input
          type="text"
          v-model="helpStore.searchQuery"
          placeholder="Search policies, CLI flags, runbooks, or diagnostics..."
          class="w-full bg-carbon-canvas border border-carbon-border rounded-xl pl-9 pr-8 py-2 text-xs text-white placeholder-slate-500 font-mono focus:outline-none focus:border-electric-blue"
        />
        <i class="pi pi-search text-slate-500 absolute left-3 top-3 text-xs pointer-events-none"></i>
        <button
          v-if="helpStore.searchQuery"
          @click="helpStore.searchQuery = ''"
          class="absolute right-2.5 top-2.5 text-slate-400 hover:text-white text-xs cursor-pointer"
        >
          <i class="pi pi-times"></i>
        </button>
      </div>

      <!-- Context Bar & Scope Filter -->
      <div class="flex items-center justify-between text-xs font-mono">
        <div class="flex items-center gap-1.5 text-slate-400 truncate">
          <span>Scope:</span>
          <span class="px-2 py-0.5 rounded bg-carbon-elevated text-electric-blue border border-carbon-border font-bold truncate">
            {{ helpStore.contextTitle }}
          </span>
        </div>
        <button
          type="button"
          @click="helpStore.toggleContextOnly()"
          :class="helpStore.contextOnly ? 'bg-electric-blue text-black font-bold' : 'bg-carbon-canvas text-slate-400 border border-carbon-border hover:text-white'"
          class="px-2 py-0.5 rounded text-[11px] transition-colors cursor-pointer"
        >
          {{ helpStore.contextOnly ? 'Scoped' : 'Show All' }}
        </button>
      </div>

      <!-- Tab Navigation -->
      <div class="flex items-center gap-1 pt-1 overflow-x-auto text-xs font-mono">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          @click="helpStore.setTab(t.id)"
          :class="helpStore.activeTab === t.id ? 'bg-electric-blue text-black font-bold shadow' : 'bg-carbon-canvas text-slate-400 hover:text-slate-200 border border-carbon-border'"
          class="px-3 py-1.5 rounded-lg transition-all flex items-center gap-1.5 cursor-pointer whitespace-nowrap"
        >
          <i :class="t.icon" class="text-xs"></i>
          <span>{{ t.label }}</span>
        </button>
      </div>
    </div>

    <!-- Body Content -->
    <div class="flex-1 overflow-y-auto p-4 space-y-4 font-mono text-xs">
      <!-- TAB 1: DIAGNOSTICS -->
      <div v-if="helpStore.activeTab === 'diagnostics'" class="space-y-4">
        <!-- Diagnostic Action Card -->
        <div class="bg-carbon-surface/80 rounded-xl border border-carbon-border p-4 space-y-3">
          <div class="flex items-center justify-between gap-3 flex-wrap">
            <div class="flex items-center gap-2.5">
              <div
                class="w-3 h-3 rounded-full shrink-0"
                :class="!helpStore.diagnostics || helpStore.diagnostics.status === 'ok' ? 'bg-emerald-400 shadow-sm' : (helpStore.diagnostics.status === 'warning' ? 'bg-amber-400' : 'bg-rose-400')"
              ></div>
              <div>
                <h3 class="text-xs font-bold text-white uppercase tracking-wider">
                  {{ helpStore.diagnostics ? (helpStore.diagnostics.status === 'ok' ? 'All Systems Healthy' : (helpStore.diagnostics.status === 'warning' ? 'System Warnings' : 'Issues Detected')) : 'Diagnostic Ready' }}
                </h3>
                <p class="text-[10px] text-slate-400">
                  {{ helpStore.diagnostics ? `Evaluated ${helpStore.diagnostics.checks?.length || 0} checks` : 'Run a full system self-diagnostic audit' }}
                </p>
              </div>
            </div>

            <button
              type="button"
              @click="helpStore.fetchDiagnostics()"
              :disabled="helpStore.loading"
              class="px-3 py-1.5 rounded-lg bg-electric-blue hover:bg-sky-400 text-black font-bold text-xs transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
            >
              <i class="pi" :class="helpStore.loading ? 'pi-spin pi-spinner' : 'pi-refresh'"></i>
              <span>{{ helpStore.loading ? 'Auditing...' : 'Run Diagnostics' }}</span>
            </button>
          </div>

          <!-- Summary Badges -->
          <div v-if="helpStore.diagnostics" class="pt-3 border-t border-carbon-border/60 grid grid-cols-2 sm:grid-cols-4 gap-2 text-center text-[11px]">
            <div class="p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
              <div class="text-emerald-400 font-bold">{{ helpStore.diagnostics.summary?.pass ?? 0 }}</div>
              <div class="text-[9px] text-slate-500 uppercase">Passing</div>
            </div>
            <div class="p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
              <div class="text-amber-400 font-bold">{{ helpStore.diagnostics.summary?.warning ?? 0 }}</div>
              <div class="text-[9px] text-slate-500 uppercase">Warnings</div>
            </div>
            <div class="p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
              <div class="text-rose-400 font-bold">{{ helpStore.diagnostics.summary?.fail ?? 0 }}</div>
              <div class="text-[9px] text-slate-500 uppercase">Failures</div>
            </div>
            <div class="p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
              <div class="text-slate-300 font-bold">{{ helpStore.diagnostics.system_info?.uptime_sec ? Math.floor(helpStore.diagnostics.system_info.uptime_sec / 60) + 'm' : '--' }}</div>
              <div class="text-[9px] text-slate-500 uppercase">Uptime</div>
            </div>
          </div>
        </div>

        <!-- Checks List -->
        <div class="space-y-2">
          <div
            v-for="check in helpStore.filteredChecks"
            :key="check.id"
            class="p-3 rounded-xl border bg-carbon-surface/60 space-y-1.5"
            :class="check.status === 'pass' ? 'border-carbon-border' : (check.status === 'warning' ? 'border-amber-500/40' : 'border-rose-500/40')"
          >
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2 truncate">
                <span
                  class="text-[9px] uppercase px-1.5 py-0.2 rounded font-bold"
                  :class="check.status === 'pass' ? 'bg-emerald-950 text-emerald-400' : (check.status === 'warning' ? 'bg-amber-950 text-amber-400' : 'bg-rose-950 text-rose-400')"
                >
                  {{ check.status }}
                </span>
                <span class="font-bold text-white text-xs truncate">{{ check.name }}</span>
              </div>
              <span class="text-[10px] text-slate-500 uppercase font-mono">{{ check.category }}</span>
            </div>
            <p class="text-[11px] text-slate-300">{{ check.message }}</p>
            <p v-if="check.detail" class="text-[10px] text-slate-400 bg-carbon-canvas p-1.5 rounded border border-carbon-border/50">
              {{ check.detail }}
            </p>
          </div>
        </div>
      </div>

      <!-- TAB 2: SAFETY POLICIES -->
      <div v-else-if="helpStore.activeTab === 'safety'" class="space-y-3">
        <div
          v-for="p in helpStore.filteredPolicies"
          :key="p.id"
          class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-2"
        >
          <div class="flex items-center justify-between gap-2">
            <h4 class="font-bold text-white text-xs">{{ p.title }}</h4>
            <span class="text-[9px] font-bold uppercase px-1.5 py-0.5 rounded bg-electric-blue/10 text-electric-blue border border-electric-blue/30 shrink-0">
              {{ p.tag }}
            </span>
          </div>
          <p class="text-[11px] text-slate-300 leading-relaxed">{{ p.description }}</p>
          <div class="p-2 rounded bg-carbon-canvas border border-carbon-border/60 text-[10px] text-slate-400">
            <span class="text-electric-blue font-bold">Action:</span> {{ p.action }}
          </div>
        </div>
      </div>

      <!-- TAB 3: FLAG CATALOG -->
      <div v-else-if="helpStore.activeTab === 'manual'" class="space-y-3">
        <div
          v-for="f in helpStore.filteredFlags"
          :key="f.flag"
          class="p-3.5 bg-carbon-surface border border-carbon-border rounded-xl space-y-2"
        >
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 truncate">
              <span class="font-bold text-electric-blue text-xs font-mono">{{ f.flag }}</span>
              <span class="text-white text-xs truncate">{{ f.name }}</span>
            </div>
            <span class="text-[9px] font-bold uppercase px-1.5 py-0.5 rounded bg-carbon-elevated text-slate-400 border border-carbon-border shrink-0">
              {{ f.category }}
            </span>
          </div>

          <p class="text-[11px] text-slate-300 leading-relaxed">{{ f.description }}</p>

          <div v-if="f.caveat" class="p-2 rounded bg-amber-500/10 border border-amber-500/20 text-[10px] text-amber-300">
            <span class="font-bold">Notice:</span> {{ f.caveat }}
          </div>

          <!-- Example Command with Copy -->
          <div class="p-2 rounded bg-carbon-canvas border border-carbon-border/60 flex items-center justify-between gap-2 text-[10px]">
            <span class="text-slate-400 font-mono truncate select-all">{{ f.example }}</span>
            <div class="flex items-center gap-1 shrink-0">
              <button
                type="button"
                @click="copyText(f.example, f.flag)"
                class="text-slate-400 hover:text-white cursor-pointer px-1 py-0.5 rounded hover:bg-carbon-elevated"
                title="Copy example"
              >
                <i class="pi" :class="copiedKey === f.flag ? 'pi-check text-emerald-400' : 'pi-copy'"></i>
              </button>
              <a
                v-if="f.docUrl"
                :href="f.docUrl"
                target="_blank"
                rel="noreferrer"
                class="text-slate-400 hover:text-electric-blue px-1 py-0.5 rounded hover:bg-carbon-elevated"
                title="Open documentation"
              >
                <i class="pi pi-external-link"></i>
              </a>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 4: RUNBOOKS -->
      <div v-else-if="helpStore.activeTab === 'runbooks'" class="space-y-3">
        <div
          v-for="r in helpStore.filteredRunbooks"
          :key="r.id"
          class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3"
        >
          <div class="flex items-center justify-between gap-2">
            <h4 class="font-bold text-white text-xs">{{ r.title }}</h4>
            <span class="text-[9px] font-bold uppercase px-1.5 py-0.5 rounded bg-carbon-elevated text-electric-blue border border-carbon-border shrink-0">
              {{ r.category }}
            </span>
          </div>

          <p class="text-[11px] text-slate-300">{{ r.summary }}</p>

          <!-- Step-by-Step Resolution List -->
          <div class="space-y-1.5 pt-1 border-t border-carbon-border/50">
            <div class="text-[10px] font-bold text-slate-400 uppercase tracking-wider">Resolution Steps</div>
            <ol class="space-y-1 text-[11px] text-slate-300 list-decimal list-inside leading-relaxed">
              <li v-for="(step, sIdx) in r.steps" :key="sIdx">
                {{ step }}
              </li>
            </ol>
          </div>

          <!-- Remediation Command -->
          <div v-if="r.command" class="p-2 rounded bg-carbon-canvas border border-carbon-border/60 flex items-center justify-between gap-2 text-[10px]">
            <span class="text-emerald-400 font-mono truncate select-all">{{ r.command }}</span>
            <button
              type="button"
              @click="copyText(r.command, r.id)"
              class="text-slate-400 hover:text-white cursor-pointer px-1 py-0.5 rounded hover:bg-carbon-elevated shrink-0"
              title="Copy remediation command"
            >
              <i class="pi" :class="copiedKey === r.id ? 'pi-check text-emerald-400' : 'pi-copy'"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </Drawer>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Drawer from 'primevue/drawer'
import { useHelpStore, type HelpTab } from '../stores/help'

const helpStore = useHelpStore()
const copiedKey = ref<string | null>(null)

const tabs: Array<{ id: HelpTab; label: string; icon: string }> = [
  { id: 'diagnostics', label: 'Diagnostics', icon: 'pi pi-heart' },
  { id: 'safety', label: 'Safety Policies', icon: 'pi pi-shield' },
  { id: 'manual', label: 'Flag Catalog', icon: 'pi pi-tags' },
  { id: 'runbooks', label: 'Runbooks', icon: 'pi pi-book' },
]

async function copyText(text: string, key: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = key
    setTimeout(() => {
      if (copiedKey.value === key) copiedKey.value = null
    }, 2000)
  } catch (e) {
    console.error('Failed to copy text:', e)
  }
}
</script>
