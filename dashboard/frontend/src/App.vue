<script setup lang="ts">
import { ref } from 'vue'

const title = ref('Bisync Hub Cockpit')
</script>

<template>
  <div class="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center">
    <div class="text-center p-8 bg-slate-900 border border-slate-800 rounded-lg shadow-xl">
      <h1 class="text-2xl font-bold text-indigo-400 mb-2">{{ title }}</h1>
      <p class="text-slate-400 text-sm">Next-Generation FSM Storage Control Plane</p>
    </div>
  </div>
</template>
