import { computed, ref } from 'vue'
import type { ThemeMode } from './types'

function readPreference(key: string): string | null {
  if (typeof window === 'undefined') return null
  try { return window.localStorage.getItem(key) } catch { return null }
}

function writePreference(key: string, value: string) {
  if (typeof window === 'undefined') return
  try { window.localStorage.setItem(key, value) } catch { /* preferences are best effort */ }
}

function readTheme(): ThemeMode {
  const value = readPreference('simpletools-theme')
  return value === 'light' || value === 'dark' || value === 'system' ? value : 'system'
}

export const themeMode = ref<ThemeMode>(readTheme())
export const systemDark = ref(typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches)
export const isDark = computed(() => themeMode.value === 'dark' || themeMode.value === 'system' && systemDark.value)

let initialized = false
export function applyTheme() {
  if (typeof document !== 'undefined') document.documentElement.classList.toggle('dark', isDark.value)
}

export function initPreferences() {
  applyTheme()
  if (initialized || typeof window === 'undefined') return
  initialized = true
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const onChange = (event: MediaQueryListEvent) => { systemDark.value = event.matches; applyTheme() }
  if (media.addEventListener) media.addEventListener('change', onChange)
  else media.addListener(onChange)
}

export function setThemeMode(value: ThemeMode) {
  themeMode.value = value
  writePreference('simpletools-theme', value)
  applyTheme()
}
