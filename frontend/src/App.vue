<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Clock3, Download, FileImage, FileOutput, FileText, Languages, Loader2, Moon, QrCode, Settings, Stamp, Sun, X, Zap } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useWorkspaceStore } from './stores/workspace'
import type { ToolId } from './types'
import type { UpdateInfo, UpdateProgress } from './types'
import { wailsService } from './services/wails'
import { initPreferences, isDark, setThemeMode } from './preferences'
import ToolNav from './components/ToolNav.vue'

const route = useRoute(); const router = useRouter(); const store = useWorkspaceStore(); const { t, locale } = useI18n()
const dark = isDark
const updateInfo = ref<UpdateInfo>()
const updateProgress = ref<UpdateProgress>()
const hasActiveWork = computed(() => store.running || store.files.some(item => item.status === 'queued' || item.status === 'processing'))
const updateError = ref('')
let stopUpdateAvailable: () => void = () => undefined
let stopUpdateProgress: () => void = () => undefined
const tools = computed(() => [
  { id: 'convert' as ToolId, icon: FileOutput, label: t('convert'), detail: t('convertDesc') },
  { id: 'compress' as ToolId, icon: FileImage, label: t('compress'), detail: t('compressDesc') },
  { id: 'watermark' as ToolId, icon: Stamp, label: t('watermark'), detail: t('watermarkDesc') },
  { id: 'qrcode' as ToolId, icon: QrCode, label: t('qrcode'), detail: t('qrcodeDesc') },
  { id: 'pdf' as ToolId, icon: FileText, label: t('pdf'), detail: t('pdfDesc') },
])
watch(() => route.path, (path) => {
  const tool = path.slice(1) as ToolId
  if (['convert', 'compress', 'watermark', 'qrcode', 'pdf'].includes(tool)) store.setTool(tool)
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
function navigate(id: ToolId) { store.setTool(id); router.push(`/${id}`) }
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
      <div class="sidebar-section-label">{{ t('nav') }}</div>
      <nav class="tool-nav">
        <ToolNav :items="tools" :active-path="route.path" @navigate="navigate" />
      </nav>
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
        <div v-if="hasActiveWork" class="global-progress" role="status"
          :aria-label="`${t('processing')} ${store.progress}%`"><span class="global-progress-track"><i
              :style="{ width: `${store.progress}%` }"></i></span><strong>{{ store.progress }}%</strong></div>
        <div class="topbar-actions"><button v-if="updateInfo?.available" class="update-button"
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
        <ToolNav :items="tools" :active-path="route.path" mobile @navigate="navigate" />
        <button class="mobile-tool-link" :class="{ active: route.path === '/recent' }" @click="router.push('/recent')">
          <Clock3 :size="16" /><span>{{ t('recent') }}</span>
        </button>
        <button class="mobile-tool-link" :class="{ active: route.path === '/settings' }"
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
