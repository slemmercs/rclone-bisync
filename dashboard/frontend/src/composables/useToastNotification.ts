import { useToast } from 'primevue/usetoast'

export interface ToastOptions {
  summary?: string
  life?: number
}

export function useToastNotification() {
  const toast = useToast()

  function success(detail: string, summary: string = 'Success', life: number = 3000) {
    toast.add({ severity: 'success', summary, detail, life })
  }

  function error(detail: string, summary: string = 'Error', life: number = 5000) {
    toast.add({ severity: 'error', summary, detail, life })
  }

  function warn(detail: string, summary: string = 'Warning', life: number = 4000) {
    toast.add({ severity: 'warn', summary, detail, life })
  }

  function info(detail: string, summary: string = 'Information', life: number = 3000) {
    toast.add({ severity: 'info', summary, detail, life })
  }

  return {
    toast,
    success,
    error,
    warn,
    info,
  }
}
