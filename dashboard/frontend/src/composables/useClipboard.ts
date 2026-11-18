import { ref } from 'vue'

export function useClipboard() {
  const copied = ref<boolean>(false)
  const error = ref<Error | null>(null)
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  async function copyToClipboard(text: string, durationMs: number = 2000): Promise<boolean> {
    error.value = null
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        const textArea = document.createElement('textarea')
        textArea.value = text
        textArea.style.position = 'fixed'
        textArea.style.left = '-999999px'
        textArea.style.top = '-999999px'
        document.body.appendChild(textArea)
        textArea.focus()
        textArea.select()
        const successful = document.execCommand('copy')
        document.body.removeChild(textArea)
        if (!successful) throw new Error('execCommand copy failed')
      }

      copied.value = true
      if (timeoutId) clearTimeout(timeoutId)
      timeoutId = setTimeout(() => {
        copied.value = false
        timeoutId = null
      }, durationMs)
      return true
    } catch (err: any) {
      console.error('Failed to copy to clipboard:', err)
      error.value = err instanceof Error ? err : new Error(String(err))
      copied.value = false
      return false
    }
  }

  return {
    copied,
    error,
    copyToClipboard,
  }
}
