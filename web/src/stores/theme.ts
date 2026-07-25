import { computed, ref } from 'vue'

export type ThemePreference = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'kejilion-panel-theme'
const preference = ref<ThemePreference>('system')
let mediaQuery: MediaQueryList | undefined

function isThemePreference(value: string | null): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

function resolvedTheme(): 'light' | 'dark' {
  if (preference.value !== 'system') return preference.value
  return mediaQuery?.matches ? 'dark' : 'light'
}

function applyTheme(): void {
  document.documentElement.dataset.theme = resolvedTheme()
  document.documentElement.style.colorScheme = resolvedTheme()
}

export function initializeTheme(): void {
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  const stored = localStorage.getItem(STORAGE_KEY)
  if (isThemePreference(stored)) preference.value = stored
  mediaQuery.addEventListener('change', applyTheme)
  applyTheme()
}

export function useTheme() {
  const setTheme = (value: ThemePreference) => {
    preference.value = value
    localStorage.setItem(STORAGE_KEY, value)
    applyTheme()
  }

  return {
    preference: computed(() => preference.value),
    resolved: computed(resolvedTheme),
    setTheme,
  }
}
