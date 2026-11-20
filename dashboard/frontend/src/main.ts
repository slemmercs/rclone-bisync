import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import { CarbonPreset } from './theme/aura-preset'
import App from './App.vue'
import './style.css'
import 'primeicons/primeicons.css'

const app = createApp(App)
app.use(createPinia())
app.use(PrimeVue, {
  theme: {
    preset: CarbonPreset,
    options: {
      darkModeSelector: '.dark',
    }
  }
})
app.use(ToastService)

app.mount('#app')


