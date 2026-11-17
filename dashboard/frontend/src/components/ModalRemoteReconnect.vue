<script setup lang="ts">
import Dialog from 'primevue/dialog'
import { useRemotesStore } from '../stores/remotes'

const remotesStore = useRemotesStore()

function handleClose() {
  remotesStore.reconnectModal.show = false
}

async function handleCancel() {
  if (remotesStore.reconnectModal.remote) {
    await remotesStore.cancelReconnect(remotesStore.reconnectModal.remote)
  } else {
    await remotesStore.stopOAuth()
  }
}
</script>

<template>
  <Dialog
    :visible="remotesStore.reconnectModal.show"
    @update:visible="(val) => remotesStore.reconnectModal.show = val"
    modal
    :style="{ width: '28rem', maxWidth: '95vw' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border rounded-2xl text-slate-100 shadow-2xl p-5' },
      header: { class: 'border-b border-carbon-border pb-3 flex items-center justify-between' },
      content: { class: 'pt-3 space-y-4 text-center' },
    }"
  >
    <template #header>
      <div class="flex items-center gap-2">
        <i class="pi pi-sync text-electric-blue text-sm"></i>
        <h3 class="text-sm font-bold text-white">OAuth Re-Authentication</h3>
      </div>
    </template>

    <div class="space-y-4">
      <!-- Status Icon -->
      <div
        class="mx-auto w-12 h-12 rounded-2xl flex items-center justify-center"
        :class="
          remotesStore.reconnectModal.error
            ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
            : remotesStore.reconnectModal.loading
            ? 'bg-electric-blue/10 text-electric-blue border border-electric-blue/20'
            : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
        "
      >
        <i
          v-if="remotesStore.reconnectModal.loading"
          class="pi pi-spin pi-spinner text-xl"
        ></i>
        <i
          v-else-if="remotesStore.reconnectModal.error"
          class="pi pi-exclamation-circle text-xl"
        ></i>
        <i
          v-else
          class="pi pi-check text-xl"
        ></i>
      </div>

      <!-- Status Header -->
      <div>
        <h4 class="text-base font-bold text-white font-mono">
          {{
            remotesStore.reconnectModal.loading
              ? 'Authorizing in Browser...'
              : remotesStore.reconnectModal.error
              ? 'Reconnect Failed'
              : 'Re-Authentication Successful'
          }}
        </h4>
        <p class="text-xs text-slate-400 mt-1 font-mono">
          Remote: <span class="text-white font-semibold">{{ remotesStore.reconnectModal.remote }}</span>
        </p>
      </div>

      <!-- Detailed Info & Fallback OAuth URL -->
      <div class="text-xs text-slate-300 bg-carbon-bg p-3.5 rounded-xl border border-carbon-border space-y-2 text-left font-mono">
        <p class="leading-relaxed">
          {{
            remotesStore.reconnectModal.msg ||
            remotesStore.reconnectModal.error ||
            'Please grant access in the browser window that opened.'
          }}
        </p>

        <div v-if="remotesStore.reconnectModal.authUrl" class="pt-2 border-t border-carbon-border/80">
          <span class="text-[11px] text-slate-400 block mb-1">If the browser did not open automatically:</span>
          <a
            :href="remotesStore.reconnectModal.authUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-electric-blue hover:underline text-[11px] break-all inline-flex items-center gap-1"
          >
            <span>Open OAuth Authorization Page</span>
            <i class="pi pi-external-link text-[10px]"></i>
          </a>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-center gap-2 pt-2">
        <button
          v-if="remotesStore.reconnectModal.loading"
          type="button"
          @click="handleCancel"
          class="px-4 py-2 bg-carbon-elevated hover:bg-slate-800 text-slate-300 border border-carbon-border rounded-xl text-xs font-semibold transition-colors cursor-pointer"
        >
          Cancel Authorization
        </button>
        <button
          v-else
          type="button"
          @click="handleClose"
          class="px-5 py-2 bg-electric-blue hover:bg-sky-400 text-black rounded-xl text-xs font-bold font-mono transition-colors cursor-pointer shadow"
        >
          Done
        </button>
      </div>
    </div>
  </Dialog>
</template>
