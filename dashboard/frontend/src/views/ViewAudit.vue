<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import { useLogsStore } from '../stores/logs'
import { useProfilesStore } from '../stores/profiles'
import type { AuditLogRecord } from '../types/models'

const logsStore = useLogsStore()
const profilesStore = useProfilesStore()

const searchQuery = ref<string>('')
const selectedProfile = ref<string>('all')

onMounted(() => {
  logsStore.fetchAuditLogs()
})

const filteredRecords = computed<AuditLogRecord[]>(() => {
  return logsStore.auditLogs.filter((rec) => {
    if (selectedProfile.value !== 'all' && rec.profile_id !== selectedProfile.value) {
      return false
    }
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      const matchAction = rec.action?.toLowerCase().includes(q)
      const matchMsg = rec.msg?.toLowerCase().includes(q)
      const matchHash = rec.hash?.toLowerCase().includes(q)
      const matchActor = rec.actor?.toLowerCase().includes(q)
      if (!matchAction && !matchMsg && !matchHash && !matchActor) {
        return false
      }
    }
    return true
  })
})

function formatTimestamp(ts?: string): string {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString()
  } catch {
    return ts
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- Audit Trail Header -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-base font-semibold text-carbon-50 tracking-tight">Cryptographic Audit Ledger</h2>
        <p class="text-xs text-carbon-400">Tamper-evident blockchain-style hash chain verifying all engine actions and user operations.</p>
      </div>

      <!-- Controls -->
      <div class="flex items-center gap-2">
        <select
          v-model="selectedProfile"
          class="bg-carbon-900 border border-carbon-700 text-xs text-carbon-300 rounded px-2.5 py-1.5 focus:outline-none"
        >
          <option value="all">All Profiles</option>
          <option v-for="p in profilesStore.profiles" :key="p.id" :value="p.id">
            {{ p.name }}
          </option>
        </select>

        <div class="relative">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search audit trail..."
            class="bg-carbon-900 border border-carbon-700 text-xs text-carbon-200 rounded pl-7 pr-3 py-1.5 focus:outline-none focus:border-electric-500 placeholder:text-carbon-500"
          />
          <i class="pi pi-search absolute left-2.5 top-2.5 text-carbon-500 text-[10px]"></i>
        </div>

        <button
          @click="logsStore.fetchAuditLogs()"
          class="px-3 py-1.5 rounded bg-carbon-800 border border-carbon-700 text-carbon-300 hover:text-white text-xs flex items-center gap-1.5"
        >
          <i :class="logsStore.loading ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" class="text-xs"></i>
          <span>Refresh</span>
        </button>
      </div>
    </div>

    <!-- Audit DataTable -->
    <div class="bg-carbon-900 border border-carbon-700 rounded-lg overflow-hidden shadow-sm">
      <DataTable
        :value="filteredRecords"
        :paginator="true"
        :rows="15"
        class="text-xs"
        responsiveLayout="scroll"
        dataKey="seq_id"
      >
        <template #empty>
          <div class="p-8 text-center text-carbon-500 text-xs">
            No audit ledger records found.
          </div>
        </template>

        <Column field="seq_id" header="Seq #" :sortable="true" style="width: 80px">
          <template #body="{ data }">
            <span class="font-mono text-carbon-400 font-semibold">#{{ data.seq_id }}</span>
          </template>
        </Column>

        <Column field="time" header="Timestamp" :sortable="true" style="width: 170px">
          <template #body="{ data }">
            <span class="font-mono text-carbon-400 text-[11px]">{{ formatTimestamp(data.time) }}</span>
          </template>
        </Column>

        <Column field="action" header="Action" :sortable="true" style="width: 140px">
          <template #body="{ data }">
            <!-- Steel Pill Token: #1E222B bg, #333947 border, #CBD5E1 text -->
            <span class="inline-flex items-center px-2 py-0.5 rounded font-mono text-[11px] font-semibold bg-[#1E222B] border border-[#333947] text-[#CBD5E1]">
              {{ data.action }}
            </span>
          </template>
        </Column>

        <Column field="profile_id" header="Profile" style="width: 120px">
          <template #body="{ data }">
            <span class="font-mono text-carbon-300">{{ data.profile_id || '--' }}</span>
          </template>
        </Column>

        <Column field="actor" header="Actor" style="width: 100px">
          <template #body="{ data }">
            <span class="text-carbon-400">{{ data.actor || 'system' }}</span>
          </template>
        </Column>

        <Column field="msg" header="Details">
          <template #body="{ data }">
            <span class="text-carbon-200">{{ data.msg }}</span>
          </template>
        </Column>

        <Column header="Hash Verification" style="width: 160px">
          <template #body="{ data }">
            <div class="font-mono text-[10px] text-carbon-500 flex items-center gap-1.5" :title="`Hash: ${data.hash}\nPrev: ${data.prev_hash}`">
              <i class="pi pi-verified text-emerald-400 text-[10px]"></i>
              <span class="truncate max-w-[110px]">{{ data.hash ? data.hash.slice(0, 10) + '...' : '--' }}</span>
            </div>
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>
