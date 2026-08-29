<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { AlertCircle, CheckCircle2, ChevronDown, CircleSlash, FileImage, FolderOpen, FolderTree, Loader2, Plus, RotateCcw, Square, Stamp, Trash2, Upload, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import Button from '../components/ui/Button.vue'
import WatermarkComparison from '../components/WatermarkComparison.vue'
import { wailsService } from '../services/wails'
import { useWorkspaceStore } from '../stores/workspace'
import type { QueueFile, WatermarkOptions, WatermarkPosition } from '../types'
import { createLatestTaskScheduler } from '../latest-task'
import { applyWatermarkPreset, matchingWatermarkPreset, renderBrowserWatermarkPreview, WATERMARK_PRESETS, type WatermarkPreset } from '../watermark'

const { t } = useI18n()
const store = useWorkspaceStore()
const input = ref<HTMLInputElement>()
const dragging = ref(false)
const selectedId = ref('')
const split = ref(50)
const beforeUrl = ref('')
const afterUrl = ref('')
const previewWidth = ref(0)
const previewHeight = ref(0)
const previewLoading = ref(false)
const previewError = ref('')
let dragDepth = 0
let previewFileId = ''

store.setTool('watermark')

const selectedFile = computed(() => store.files.find(item => item.id === selectedId.value) ?? store.files[0])
const activePreset = computed(() => matchingWatermarkPreset(store.settings.watermark))
const selectedFormat = computed(() => {
  const item = selectedFile.value
  if (!item) return '—'
  const extensionIndex = item.name.lastIndexOf('.')
  if (extensionIndex > 0 && extensionIndex < item.name.length - 1) return item.name.slice(extensionIndex + 1).toUpperCase()
  const subtype = item.type.match(/^image\/([^;]+)/i)?.[1]?.replace('+xml', '').replace(/^x-/, '')
  return subtype ? subtype.toUpperCase() : '—'
})
const opacityPercent = computed({
  get: () => Math.round(store.settings.watermark.opacity * 100),
  set: (value: number) => { store.settings.watermark.opacity = Math.min(1, Math.max(0.01, value / 100)) },
})
const positionOptions: Array<{ value: WatermarkPosition; label: string }> = [
  { value: 'top-left', label: 'positionTopLeft' },
  { value: 'top-center', label: 'positionTopCenter' },
  { value: 'top-right', label: 'positionTopRight' },
  { value: 'center-left', label: 'positionCenterLeft' },
  { value: 'center', label: 'positionCenter' },
  { value: 'center-right', label: 'positionCenterRight' },
  { value: 'bottom-left', label: 'positionBottomLeft' },
  { value: 'bottom-center', label: 'positionBottomCenter' },
  { value: 'bottom-right', label: 'positionBottomRight' },
]

interface PreviewTaskInput {
  item: QueueFile
  options: WatermarkOptions
}

interface PreviewTaskResult {
  fileId: string
  beforeDataUrl: string
  afterDataUrl: string
  width: number
  height: number
}

const previewScheduler = createLatestTaskScheduler<PreviewTaskInput, PreviewTaskResult>({
  debounceMs: 160,
  async task({ item, options }) {
    const result = item.path && wailsService.isNative()
      ? await wailsService.previewWatermark(item.path, options, 960)
      : item.file
        ? await renderBrowserWatermarkPreview(item.file, options, 960)
        : undefined
    if (!result?.beforeDataUrl || !result.afterDataUrl) throw new Error('Preview payload is empty')
    return { ...result, fileId: item.id }
  },
  onSuccess(result) {
    beforeUrl.value = result.beforeDataUrl
    afterUrl.value = result.afterDataUrl
    previewWidth.value = result.width
    previewHeight.value = result.height
    previewFileId = result.fileId
    previewLoading.value = false
  },
  onError() {
    beforeUrl.value = ''
    afterUrl.value = ''
    previewFileId = ''
    previewError.value = t('previewFailed')
    previewLoading.value = false
  },
})

watch(() => store.files.map(item => item.id), (ids) => {
  if (!ids.includes(selectedId.value)) selectedId.value = ids[0] ?? ''
}, { immediate: true })

watch([
  () => selectedFile.value?.id,
  () => store.settings.watermark,
], () => schedulePreview(), { deep: true, immediate: true })

function chooseFiles(files: FileList | File[]) {
  store.addFiles(files)
  dragging.value = false
  dragDepth = 0
}

function onDragEnter() {
  dragDepth += 1
  dragging.value = true
}

function onDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragging.value = false
}

function onDrop(event: DragEvent) {
  if (event.dataTransfer?.files) chooseFiles(event.dataTransfer.files)
  else {
    dragging.value = false
    dragDepth = 0
  }
}

function onInputChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files) chooseFiles(target.files)
  target.value = ''
}

function browse() {
  if (wailsService.isNative()) void store.browseFiles()
  else input.value?.click()
}

function browseFolder() { void store.browseFolder() }
function selectFile(item: QueueFile) { selectedId.value = item.id }
function formatSize(bytes: number) { return bytes < 1024 * 1024 ? `${Math.max(1, Math.round(bytes / 1024))} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB` }

function selectPreset(preset: WatermarkPreset) {
  store.settings.watermark = applyWatermarkPreset(store.settings.watermark, preset)
}

function schedulePreview() {
  const item = selectedFile.value
  if (!item) {
    previewScheduler.cancel()
    previewFileId = ''
    beforeUrl.value = ''
    afterUrl.value = ''
    previewWidth.value = 0
    previewHeight.value = 0
    previewLoading.value = false
    previewError.value = ''
    return
  }
  if (previewFileId && previewFileId !== item.id) {
    beforeUrl.value = ''
    afterUrl.value = ''
  }
  previewLoading.value = true
  previewError.value = ''
  previewScheduler.schedule({ item, options: { ...store.settings.watermark } })
}

onMounted(() => {
  if (!wailsService.isNative()) return
  void store.loadDefaultOutputDirectory()
  OnFileDrop((_x, _y, paths) => { void store.addNativePaths(paths) }, true)
})

onUnmounted(() => {
  previewScheduler.dispose()
  if (wailsService.isNative()) OnFileDropOff()
})
</script>

<template>
  <section class="page-header">
    <div>
      <div class="eyebrow"><Stamp :size="14" /> {{ t('appName') }}</div>
      <h1>{{ t('watermark') }}</h1>
      <p>{{ t('watermarkDesc') }}</p>
    </div>
    <div class="header-stat"><span class="stat-value">{{ store.files.length }}</span><span class="stat-label">{{ t('files') }}</span></div>
  </section>

  <section class="drop-zone watermark-drop-zone" :class="{ dragging }" @dragenter.prevent="onDragEnter" @dragover.prevent="dragging = true" @dragleave.prevent="onDragLeave" @drop.prevent="onDrop" @click="browse">
    <div class="upload-icon"><Upload :size="21" /></div>
    <div class="watermark-drop-copy"><h2>{{ t('dropTitle') }}</h2><p>{{ t('dropHint') }}</p></div>
    <div class="drop-actions"><button class="secondary-button" @click.stop="browse"><Plus :size="16" /> {{ t('browse') }}</button><button class="secondary-button" @click.stop="browseFolder"><FolderTree :size="16" /> {{ t('folder') }}</button></div>
    <input ref="input" class="hidden-input" type="file" multiple accept=".png,.jpg,.jpeg,.webp,.avif,.ico,.svg,image/*" @change="onInputChange" />
  </section>

  <div class="watermark-workspace">
    <div class="watermark-main-column">
      <section class="panel watermark-preview-panel">
        <div class="panel-header watermark-preview-header">
          <div><h2>{{ t('livePreview') }}</h2><span class="muted" :title="selectedFile?.name">{{ selectedFile?.name || t('noFiles') }}</span></div>
          <div class="comparison-key" aria-hidden="true"><span><i class="before-key"></i>{{ t('beforeWatermark') }}</span><span><i class="after-key"></i>{{ t('afterWatermark') }}</span></div>
        </div>
        <div class="watermark-preview-body">
          <WatermarkComparison
            v-model="split"
            :before-url="beforeUrl"
            :after-url="afterUrl"
            :width="previewWidth"
            :height="previewHeight"
            :alt="selectedFile?.name"
            :before-label="t('beforeWatermark')"
            :after-label="t('afterWatermark')"
            :empty-label="t('watermarkPreviewEmpty')"
            :slider-label="t('compareSlider')"
            :loading-label="t('previewing')"
            :loading="previewLoading"
            :error="previewError"
          />
        </div>
      </section>

      <section class="queue-panel panel watermark-queue-panel">
        <div class="panel-header"><div><h2>{{ t('queue') }}</h2><span class="muted">{{ store.files.length }} {{ t('files') }}</span></div><button v-if="store.files.length" class="text-button" :disabled="store.running" @click="store.clearFiles"><Trash2 :size="15" /> {{ t('clear') }}</button></div>
        <div v-if="!store.files.length" class="empty-state"><div class="empty-icon"><Upload :size="20" /></div><strong>{{ t('noFiles') }}</strong><span>{{ t('noFilesHint') }}</span></div>
        <div v-else class="file-list watermark-file-list">
          <div v-for="item in store.files" :key="item.id" class="file-row watermark-file-row" :class="{ selected: item.id === selectedFile?.id }" @click="selectFile(item)">
            <button class="file-type watermark-file-select" :aria-label="`${t('selectPreview')}: ${item.name}`" :aria-pressed="item.id === selectedFile?.id" @click.stop="selectFile(item)"><img v-if="item.previewUrl" :src="item.previewUrl" :alt="item.name" /><FileImage v-else :size="18" /></button>
            <div class="file-meta"><div class="file-name" :title="item.name">{{ item.name }}</div><div class="file-sub">{{ formatSize(item.size) }}<span v-if="item.status === 'done'"> · {{ item.resultName }}</span></div><div v-if="item.status === 'processing' || item.status === 'queued'" class="progress-track"><span :style="{ width: `${item.progress}%` }"></span></div><div v-if="item.status === 'error'" class="error-copy"><AlertCircle :size="13" /> {{ item.error || t('failed') }}</div><div v-if="item.status === 'cancelled'" class="cancel-copy"><CircleSlash :size="13" /> {{ t('cancelled') }}</div><div v-if="item.warning" class="warning-copy">{{ item.warning }}</div></div>
            <div class="file-status"><Loader2 v-if="item.status === 'processing'" class="spin" :size="16" /><CheckCircle2 v-else-if="item.status === 'done'" class="success" :size="17" /><button v-else-if="item.status === 'error'" class="icon-button small" :title="t('retry')" @click.stop="store.retry(item.id)"><RotateCcw :size="15" /></button><CircleSlash v-else-if="item.status === 'cancelled'" class="cancelled" :size="16" /><span v-else class="queue-dot"></span><button class="icon-button small remove-button" :disabled="item.status === 'processing'" :title="t('remove')" @click.stop="store.removeFile(item.id)"><X :size="15" /></button></div>
          </div>
        </div>
      </section>
    </div>

    <aside class="settings-panel panel watermark-settings-panel">
      <h2>{{ t('watermarkStyle') }}</h2>
      <div class="field">
        <label for="watermark-text">{{ t('watermarkText') }}</label>
        <input id="watermark-text" v-model="store.settings.watermark.text" class="watermark-text-input" type="text" maxlength="2000" :placeholder="t('watermarkTextPlaceholder')" />
      </div>
      <div class="field">
        <label>{{ t('watermarkPreset') }}</label>
        <div class="watermark-presets">
          <button v-for="preset in WATERMARK_PRESETS" :key="preset.id" :class="{ selected: activePreset === preset.id }" :aria-pressed="activePreset === preset.id" @click="selectPreset(preset)"><span class="preset-mark" :class="`preset-${preset.id}`">Aa</span><span>{{ t(`preset${preset.id.charAt(0).toUpperCase()}${preset.id.slice(1)}`) }}</span></button>
        </div>
      </div>
      <div class="field watermark-control-grid">
        <div>
          <div class="label-with-value"><label for="watermark-size">{{ t('size') }}</label><strong>{{ store.settings.watermark.fontSize }} px</strong></div>
          <input id="watermark-size" v-model.number="store.settings.watermark.fontSize" class="range" type="range" min="12" max="240" step="1" />
        </div>
        <div>
          <div class="label-with-value"><label for="watermark-opacity">{{ t('opacity') }}</label><strong>{{ opacityPercent }}%</strong></div>
          <input id="watermark-opacity" v-model.number="opacityPercent" class="range" type="range" min="1" max="100" step="1" />
        </div>
      </div>
      <div class="field">
        <label for="watermark-color">{{ t('color') }}</label>
        <div class="watermark-color-control"><input id="watermark-color" v-model="store.settings.watermark.color" type="color" /><span>{{ store.settings.watermark.color.toUpperCase() }}</span></div>
      </div>
      <div class="field">
        <label for="watermark-font">{{ t('font') }}</label>
        <select id="watermark-font" v-model="store.settings.watermark.fontFamily" class="watermark-select"><option value="noto-sans-sc">{{ t('fontNoto') }}</option></select>
      </div>
      <div v-if="!store.settings.watermark.tile" class="field">
        <label>{{ t('position') }}</label>
        <div class="watermark-position-grid">
          <button v-for="option in positionOptions" :key="option.value" :class="{ selected: store.settings.watermark.position === option.value }" :title="t(option.label)" :aria-label="t(option.label)" :aria-pressed="store.settings.watermark.position === option.value" @click="store.settings.watermark.position = option.value"><span></span></button>
        </div>
      </div>
      <div class="field watermark-control-grid">
        <div>
          <div class="label-with-value"><label for="watermark-rotation">{{ t('rotation') }}</label><strong>{{ store.settings.watermark.rotation }}°</strong></div>
          <input id="watermark-rotation" v-model.number="store.settings.watermark.rotation" class="range" type="range" min="-180" max="180" step="1" />
        </div>
        <div>
          <div class="label-with-value"><label :for="store.settings.watermark.tile ? 'watermark-spacing' : 'watermark-margin'">{{ store.settings.watermark.tile ? t('tileSpacing') : t('margin') }}</label><strong>{{ store.settings.watermark.tile ? store.settings.watermark.spacing : store.settings.watermark.margin }} px</strong></div>
          <input v-if="store.settings.watermark.tile" id="watermark-spacing" v-model.number="store.settings.watermark.spacing" class="range" type="range" min="20" max="500" step="1" />
          <input v-else id="watermark-margin" v-model.number="store.settings.watermark.margin" class="range" type="range" min="0" max="240" step="1" />
        </div>
      </div>
      <div class="field watermark-toggle-grid">
        <label class="checkbox-row"><input v-model="store.settings.watermark.tile" type="checkbox" /><span>{{ t('tile') }}</span></label>
        <label class="checkbox-row"><input v-model="store.settings.watermark.shadow" type="checkbox" /><span>{{ t('shadow') }}</span></label>
      </div>
      <div class="field watermark-output-field">
        <label>{{ t('output') }}</label>
        <div class="directory-input"><FolderOpen :size="16" /><input v-model="store.outputDir" :placeholder="t('outputPlaceholder')" @change="store.setOutputDir(store.outputDir)" /><button :title="t('change')" @click="store.chooseOutput"><ChevronDown :size="15" /></button></div>
      </div>
      <div class="field option-field"><label class="checkbox-row"><input v-model="store.settings.recursive" type="checkbox" /><span>{{ t('includeSubfolders') }}</span></label><label class="checkbox-row"><input v-model="store.settings.preserveMetadata" type="checkbox" /><span>{{ t('preserveMetadata') }}</span></label></div>
      <div class="estimate"><span>{{ t('sourceFormat') }}</span><strong>{{ selectedFormat }}</strong></div>
      <div class="action-row"><button v-if="store.running" class="secondary-button" @click="store.cancel"><Square :size="15" /> {{ t('cancel') }}</button><Button class="primary-button" :disabled="!store.canProcess" @click="store.process"><Loader2 v-if="store.running" class="spin" :size="17" /><Stamp v-else :size="17" />{{ store.running ? t('processing') : t('applyWatermark') }}</Button></div>
    </aside>
  </div>

  <div v-if="store.completeCount && !store.running && !store.files.some(item => item.status === 'queued' || item.status === 'processing')" class="toast"><CheckCircle2 :size="17" /> {{ t('allDone') }} · {{ store.completeCount }} {{ t('files') }}<button class="icon-button small" :title="t('openFolder')" @click="store.openOutputDirectory"><FolderOpen :size="15" /></button></div>
</template>
