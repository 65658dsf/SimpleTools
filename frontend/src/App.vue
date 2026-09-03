<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Clock3, Download, Languages, Loader2, Moon, Settings, Sun, X, Zap } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useWorkspaceStore } from './stores/workspace'
import type { ToolId } from './types'
import type { UpdateInfo, UpdateProgress } from './types'
import { wailsService } from './services/wails'
import { initPreferences, isDark, setThemeMode } from './preferences'
import HomeButton from './components/HomeButton.vue'

const route = useRoute(); const router = useRouter(); const store = useWorkspaceStore(); const { t, locale } = useI18n()
const dark = isDark
const updateInfo = ref<UpdateInfo>()
const updateProgress = ref<UpdateProgress>()
const hasActiveWork = computed(() => store.running || store.files.some(item => item.status === 'queued' || item.status === 'processing'))
const lastToolPath = ref('')
const updateError = ref('')
let stopUpdateAvailable: () => void = () => undefined
let stopUpdateProgress: () => void = () => undefined
watch(() => route.path, (path) => {
  const tool = path.slice(1) as ToolId
  if (store.running && path === '/' && lastToolPath.value) {
    void router.replace(lastToolPath.value)
    return
  }
  if (!['convert', 'compress', 'watermark', 'qrcode', 'pdf'].includes(tool)) return
  if (store.running && tool !== store.activeTool) {
    const fallback = lastToolPath.value || `/${store.activeTool}`
    if (path !== fallback) void router.replace(fallback)
    return
  }
  if (store.setTool(tool)) lastToolPath.value = path
}, { immediate: true })
watch(locale, value => {
  if (typeof window === 'undefined') return
  try { window.localStorage.setItem('simpletools-language', value) } catch { /* preferences are best effort */ }
})
onMounted(() => {
  initPreferences()
  if (!wailsService.isNative()) return
  stopUpdateAvailable = wailsService.onUpdateAvailable(info => { if (info.available) updateInfo.value = info })
  stopUpdateProgress = wailsService.onUpdateProgress(progress => {
    updateProgress.value = progress
    if (progress.state === 'failed') updateError.value = t('updateFailed')
  })
  void wailsService.checkForUpdate().then(info => { if (info.available) updateInfo.value = info }).catch(() => undefined)
})
onUnmounted(() => { stopUpdateAvailable(); stopUpdateProgress() })
function toggleTheme() { setThemeMode(dark.value ? 'light' : 'dark') }
function toggleLocale() { locale.value = locale.value === 'en' ? 'zh' : 'en' }
async function installUpdate() {
  if (!updateInfo.value || updateProgress.value?.state === 'started') return
  updateError.value = ''
  updateProgress.value = { assetId: updateInfo.value.assetId ?? '', state: 'started', progress: 0 }
  try {
    await wailsService.downloadAndInstallUpdate(updateInfo.value.assetId ?? '')
  } catch (error) {
    updateError.value = error instanceof Error ? error.message : t('updateFailed')
    updateProgress.value = { assetId: updateInfo.value.assetId ?? '', state: 'failed', progress: 0 }
  }
}
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand"><span class="brand-mark">
          <Zap :size="16" />
        </span><span>{{ t('appName') }}</span></div>
      <div v-if="route.path !== '/'" class="sidebar-home">
        <HomeButton :disabled="store.running" />
      </div>
      <div class="sidebar-bottom">
        <button class="tool-link" :class="{ active: route.path === '/recent' }" @click="router.push('/recent')">
          <Clock3 :size="18" /><span>{{ t('recent') }}</span>
        </button>
        <button class="tool-link" :class="{ active: route.path === '/settings' }" @click="router.push('/settings')">
          <Settings :size="18" /><span>{{ t('preferences') }}</span>
        </button>
        <div class="sidebar-note"><span class="status-dot"></span><span>{{ t('saved') }}</span></div>
      </div>
    </aside>
    <main class="main-content">
      <header class="topbar">
        <div class="mobile-brand"><span class="brand-mark">
            <Zap :size="15" />
          </span>{{ t('appName') }}</div>
        <div
v-if="hasActiveWork" class="global-progress" role="status"
          :aria-label="`${t('processing')} ${store.progress}%`"><span class="global-progress-track"><i
              :style="{ width: `${store.progress}%` }"></i></span><strong>{{ store.progress }}%</strong></div>
        <div class="topbar-actions"><button
v-if="updateInfo?.available" class="update-button"
            :title="t('updateAvailable')" @click="installUpdate">
            <Loader2 v-if="updateProgress?.state === 'started'" class="spin" :size="16" />
            <Download v-else :size="16" /><span>{{ updateProgress?.state === 'started' ? t('updating') :
              `${t('updateAvailable')} · ${updateInfo.version}` }}</span>
          </button><button class="icon-button" :title="t('language')" @click="toggleLocale">
            <Languages :size="18" /><span class="lang-label">{{ locale === 'en' ? '中' : 'EN' }}</span>
          </button><button class="icon-button" :title="t('theme')" @click="toggleTheme">
            <Sun v-if="dark" :size="18" />
            <Moon v-else :size="18" />
          </button></div>
      </header>
      <nav class="mobile-tool-nav" :aria-label="t('nav')">
        <div v-if="route.path !== '/'" class="mobile-home">
          <HomeButton :disabled="store.running" />
        </div>
        <button class="mobile-tool-link" :class="{ active: route.path === '/recent' }" @click="router.push('/recent')">
          <Clock3 :size="16" /><span>{{ t('recent') }}</span>
        </button>
        <button
class="mobile-tool-link" :class="{ active: route.path === '/settings' }"
          @click="router.push('/settings')">
          <Settings :size="16" /><span>{{ t('preferences') }}</span>
        </button>
      </nav>
      <div v-if="updateError" class="update-error" role="status"><span>{{ updateError }}</span><button
          class="icon-button small" :title="t('dismiss')" @click="updateError = ''">
          <X :size="14" />
        </button></div>
      <div class="content-wrap">
        <RouterView />
      </div>
    </main>
  </div>
</template>
