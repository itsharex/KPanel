import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import { initializeI18n } from './i18n'
import { initializeTheme } from './stores/theme'
import { initializeDesktopMode } from './stores/desktopMode'
import './styles/main.css'

initializeTheme()
initializeDesktopMode()

async function bootstrap(): Promise<void> {
  await initializeI18n()
  createApp(App).use(router).mount('#app')
}

void bootstrap()
