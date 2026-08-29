<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { AlertCircle, CheckCircle2, ChevronDown, Download, FolderOpen, Loader2, QrCode, RotateCcw } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import Button from '../components/ui/Button.vue'
import QRCodeDecodePanel from '../components/QRCodeDecodePanel.vue'
import { createLatestTaskScheduler } from '../latest-task'
import { DEFAULT_QR_CODE, QR_CODE_ERROR_CORRECTIONS, QR_CODE_SIZES, qrCodeByteLength, qrCodeOptions, qrCodeTabForKey } from '../qrcode'
import { wailsService } from '../services/wails'
import { useWorkspaceStore } from '../stores/workspace'
import type { QRCodeTab } from '../qrcode'
import type { QRCodeErrorCorrection, QRCodeOptions, QRCodePreview } from '../types'

const { t } = useI18n()
const store = useWorkspaceStore()
const preview = ref<QRCodePreview>()
const previewLoading = ref(false)
const previewError = ref('')
const saving = ref(false)
const saveError = ref('')
const savedPath = ref('')
const activeTab = ref<QRCodeTab>('generate')
const generateTab = ref<HTMLButtonElement>()
const decodeTab = ref<HTMLButtonElement>()

store.setTool('qrcode')

const settings = computed(() => store.settings.qrCode)
const byteCount = computed(() => qrCodeByteLength(settings.value.text))
const canSave = computed(() => Boolean(settings.value.text.trim() && settings.value.fileName.trim() && !previewError.value && !saving.value))
const savedName = computed(() => savedPath.value.split(/[\\/]/).pop() ?? '')
const correctionLabels: Record<QRCodeErrorCorrection, string> = {
  low: 'L · 7%',
  medium: 'M · 15%',
  quartile: 'Q · 25%',
  high: 'H · 30%',
}

const previewScheduler = createLatestTaskScheduler<QRCodeOptions, QRCodePreview>({
  debounceMs: 140,
  task: options => wailsService.previewQRCode(options, 560),
  onSuccess(result) {
    preview.value = result
    previewLoading.value = false
  },
  onError() {
    preview.value = undefined
    previewError.value = t('qrPreviewFailed')
    previewLoading.value = false
  },
})

watch(() => store.settings.qrCode, value => {
  savedPath.value = ''
  saveError.value = ''
  previewError.value = ''
  if (!value.text.trim()) {
    previewScheduler.cancel()
    preview.value = undefined
    previewLoading.value = false
    return
  }
  previewLoading.value = true
  previewScheduler.schedule(qrCodeOptions({ ...value }))
}, { deep: true, immediate: true })

function resetColors() {
  settings.value.foreground = DEFAULT_QR_CODE.foreground
  settings.value.background = DEFAULT_QR_CODE.background
}

function selectTab(tab: QRCodeTab) {
  activeTab.value = tab
}

function handleTabKey(event: KeyboardEvent) {
  const tab = qrCodeTabForKey(activeTab.value, event.key)
  if (!tab) return
  event.preventDefault()
  selectTab(tab)
  void nextTick(() => (tab === 'generate' ? generateTab.value : decodeTab.value)?.focus())
}

async function saveQRCode() {
  if (!canSave.value) return
  saving.value = true
  saveError.value = ''
  savedPath.value = ''
  try {
    if (wailsService.isNative()) await store.loadDefaultOutputDirectory()
    savedPath.value = await wailsService.saveQRCode(qrCodeOptions({ ...settings.value }), store.outputDir, settings.value.fileName)
  } catch {
    saveError.value = t('qrSaveFailed')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (wailsService.isNative()) void store.loadDefaultOutputDirectory()
})

onUnmounted(() => previewScheduler.dispose())
</script>

<template>
  <section class="page-header">
    <div>
      <div class="eyebrow"><QrCode :size="14" /> {{ t('appName') }}</div>
      <h1>{{ t('qrcode') }}</h1>
      <p>{{ t('qrcodeDesc') }}</p>
    </div>
    <div v-if="activeTab === 'generate'" class="header-stat"><span class="stat-value">{{ settings.size }}</span><span class="stat-label">px · PNG</span></div>
  </section>

  <nav class="qr-tabs" role="tablist" :aria-label="t('qrcode')">
    <button id="qr-generate-tab" ref="generateTab" role="tab" :aria-selected="activeTab === 'generate'" aria-controls="qr-generate-panel" :tabindex="activeTab === 'generate' ? 0 : -1" :class="{ selected: activeTab === 'generate' }" @click="selectTab('generate')" @keydown="handleTabKey">{{ t('qrGenerateTab') }}</button>
    <button id="qr-decode-tab" ref="decodeTab" role="tab" :aria-selected="activeTab === 'decode'" aria-controls="qr-decode-panel" :tabindex="activeTab === 'decode' ? 0 : -1" :class="{ selected: activeTab === 'decode' }" @click="selectTab('decode')" @keydown="handleTabKey">{{ t('qrDecodeTab') }}</button>
  </nav>

  <div v-show="activeTab === 'generate'" id="qr-generate-panel" class="qr-tab-panel" role="tabpanel" aria-labelledby="qr-generate-tab">
    <div class="qr-workspace">
    <section class="panel qr-preview-panel">
      <div class="panel-header">
        <div><h2>{{ t('livePreview') }}</h2><span class="muted">{{ byteCount }} {{ t('bytes') }}</span></div>
        <span class="qr-format-badge">PNG</span>
      </div>
      <div class="qr-preview-body">
        <div class="qr-preview-surface" :class="{ empty: !preview?.dataUrl }">
          <img v-if="preview?.dataUrl" :src="preview.dataUrl" :alt="t('qrPreviewAlt')" />
          <div v-if="previewLoading" class="qr-preview-state" role="status"><Loader2 class="spin" :size="24" /><span>{{ t('qrGenerating') }}</span></div>
          <div v-else-if="previewError" class="qr-preview-state error" role="status"><AlertCircle :size="24" /><span>{{ previewError }}</span></div>
          <div v-else-if="!preview?.dataUrl" class="qr-preview-state"><QrCode :size="27" /><span>{{ t('qrEmptyPreview') }}</span></div>
        </div>
      </div>
      <div class="qr-preview-meta">
        <div><span>{{ t('qrContentLength') }}</span><strong>{{ settings.text.length }}</strong></div>
        <div><span>{{ t('qrByteLength') }}</span><strong>{{ byteCount }}</strong></div>
        <div><span>{{ t('qrCorrection') }}</span><strong>{{ correctionLabels[settings.errorCorrection] }}</strong></div>
      </div>
    </section>

    <aside class="settings-panel panel qr-settings-panel">
      <h2>{{ t('qrSettings') }}</h2>
      <div class="field">
        <div class="label-with-value"><label for="qr-text">{{ t('qrText') }}</label><strong>{{ byteCount }} {{ t('bytes') }}</strong></div>
        <textarea id="qr-text" v-model="settings.text" class="qr-textarea" maxlength="4096" :placeholder="t('qrTextPlaceholder')"></textarea>
      </div>
      <div class="field">
        <label>{{ t('qrSize') }}</label>
        <div class="segmented qr-size-segmented">
          <button v-for="size in QR_CODE_SIZES" :key="size" :class="{ selected: settings.size === size }" :aria-pressed="settings.size === size" @click="settings.size = size">{{ size }}</button>
        </div>
      </div>
      <div class="field">
        <label>{{ t('qrCorrection') }}</label>
        <div class="segmented qr-correction-segmented">
          <button v-for="level in QR_CODE_ERROR_CORRECTIONS" :key="level" :class="{ selected: settings.errorCorrection === level }" :aria-pressed="settings.errorCorrection === level" :title="t(`qrCorrection${level.charAt(0).toUpperCase()}${level.slice(1)}`)" @click="settings.errorCorrection = level">{{ correctionLabels[level] }}</button>
        </div>
      </div>
      <div class="field">
        <div class="label-with-value"><label>{{ t('qrColors') }}</label><button class="icon-button small" :title="t('resetColors')" @click="resetColors"><RotateCcw :size="14" /></button></div>
        <div class="qr-color-grid">
          <label><span>{{ t('foreground') }}</span><span class="qr-color-input"><input v-model="settings.foreground" type="color" /><code>{{ settings.foreground.toUpperCase() }}</code></span></label>
          <label><span>{{ t('background') }}</span><span class="qr-color-input"><input v-model="settings.background" type="color" /><code>{{ settings.background.toUpperCase() }}</code></span></label>
        </div>
      </div>
      <div class="field qr-output-field">
        <label for="qr-file-name">{{ t('fileName') }}</label>
        <div class="qr-file-name"><input id="qr-file-name" v-model="settings.fileName" maxlength="160" /><span>.png</span></div>
      </div>
      <div class="field">
        <label>{{ t('output') }}</label>
        <div class="directory-input"><FolderOpen :size="16" /><input v-model="store.outputDir" :placeholder="t('outputPlaceholder')" @change="store.setOutputDir(store.outputDir)" /><button :title="t('change')" @click="store.chooseOutput"><ChevronDown :size="15" /></button></div>
      </div>
      <div v-if="saveError" class="qr-save-status error" role="status"><AlertCircle :size="15" /> {{ saveError }}</div>
      <div v-else-if="savedPath" class="qr-save-status success" role="status"><CheckCircle2 :size="15" /><span :title="savedPath">{{ savedName }}</span><button class="icon-button small" :title="t('openFolder')" @click="store.openOutputDirectory"><FolderOpen :size="14" /></button></div>
      <div class="action-row qr-action-row">
        <Button class="primary-button" :disabled="!canSave" @click="saveQRCode"><Loader2 v-if="saving" class="spin qr-button-spinner" :size="17" /><Download v-else :size="17" />{{ saving ? t('qrSaving') : t('qrSave') }}</Button>
      </div>
    </aside>
    </div>
  </div>
  <QRCodeDecodePanel v-show="activeTab === 'decode'" id="qr-decode-panel" class="qr-tab-panel" role="tabpanel" aria-labelledby="qr-decode-tab" />
</template>
