import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const activeTab = ref<string>('fleet')

  function switchTab(tab: string) {
    activeTab.value = tab
  }

  return {
    activeTab,
    switchTab,
  }
})
