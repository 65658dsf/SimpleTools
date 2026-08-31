<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  ChevronDown,
  CircleSlash,
  FileImage,
  FileText,
  FolderOpen,
  Loader2,
  Plus,
  RotateCcw,
  Trash2,
  Upload,
  X,
  FolderTree,
  Square,
} from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useWorkspaceStore, type QueueFilter } from '../stores/workspace'
import { wailsService } from '../services/wails'
import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import Button from '../components/ui/Button.vue'

const { t } = useI18n()
const store = useWorkspaceStore()
const input = ref<HTMLInputElement>()
const dragging = ref(false)
let dragDepth = 0
const metadata = computed(() =>
  store.activeTool === 'convert'
    ? { title: t('convert'), desc: t('convertDesc'), accepts: t('imageTypes'), icon: FileImage }
    : store.activeTool === 'compress'
      ? { title: t('compress'), desc: t('compressDesc'), accepts: t('imageTypes'), icon: FileImage }
      : { title: t('pdf'), desc: t('pdfDesc'), accepts: t('pdfType'), icon: FileText },
)
const formatOptions = ['webp', 'jpg', 'png', 'avif', 'ico', 'svg']
const dpiOptions = [72, 150, 300, 600]
const queueFilters = [
  { value: 'all', label: 'all' },
  { value: 'queued', label: 'queued' },
  { value: 'processing', label: 'processingFilter' },
  { value: 'done', label: 'doneFilter' },
  { value: 'error', label: 'errorFilter' },
  { value: 'cancelled', label: 'cancelledFilter' },
] as const
const format = computed({
  get: () => store.settings.format,
  set: (value: string) => {
    store.settings.format = value
  },
})
const quality = computed({
  get: () => store.settings.quality,
  set: (value: number) => {
    store.settings.quality = value
  },
})
const dpi = computed({
  get: () => store.settings.dpi,
  set: (value: number) => {
    store.settings.dpi = value
  },
})
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
function browseFolder() {
  void store.browseFolder()
}
function onKeydown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  const editing =
    target?.tagName === 'INPUT' ||
    target?.tagName === 'TEXTAREA' ||
    target?.tagName === 'SELECT' ||
    target?.isContentEditable
  const modifier = event.ctrlKey || event.metaKey
  if (event.key === 'Escape' && store.running) {
    event.preventDefault()
    void store.cancel()
    return
  }
  if (!modifier || editing) return
  const key = event.key.toLowerCase()
  if (key === 'o' && event.shiftKey) {
    event.preventDefault()
    void store.chooseOutput()
  } else if (key === 'o') {
    event.preventDefault()
    browse()
  } else if (event.key === 'Enter') {
    event.preventDefault()
    void store.process()
  }
}
function formatSize(bytes: number) {
  const normalized = Number.isFinite(bytes) ? Math.max(0, bytes) : 0
  if (normalized < 1024) return `${Math.round(normalized)} B`
  if (normalized < 1024 * 1024)
    return `${(normalized / 1024).toFixed(normalized < 10 * 1024 ? 1 : 0)} KB`
  if (normalized < 1024 * 1024 * 1024)
    return `${(normalized / 1024 / 1024).toFixed(normalized < 10 * 1024 * 1024 ? 1 : 0)} MB`
  return `${(normalized / 1024 / 1024 / 1024).toFixed(1)} GB`
}
const LARGE_IMAGE_BYTES = 64 * 1024 * 1024
function compressionComparison(item: { originalBytes?: number; compressedBytes?: number }) {
  if (
    item.originalBytes === undefined ||
    item.compressedBytes === undefined ||
    item.originalBytes <= 0
  )
    return ''
  const savings = Math.round((1 - item.compressedBytes / item.originalBytes) * 100)
  const sign = savings >= 0 ? '-' : '+'
  const label = savings >= 0 ? t('compressionSavings') : t('compressionLarger')
  return `${formatSize(item.originalBytes)} → ${formatSize(item.compressedBytes)} (${sign}${Math.abs(savings)}% ${label})`
}
function filterCount(value: QueueFilter) {
  return value === 'all'
    ? store.files.length
    : store.files.filter((item) => item.status === value).length
}
function rowIcon(type: string) {
  return type === 'application/pdf' ? FileText : FileImage
}
onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  if (!wailsService.isNative()) return
  void store.loadDefaultOutputDirectory()
  OnFileDrop((_x, _y, paths) => {
    void store.addNativePaths(paths)
  }, true)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  if (wailsService.isNative()) OnFileDropOff()
})
</script>

<template>
  <section class="page-header">
    <div>
      <div class="eyebrow"><component :is="metadata.icon" :size="14" /> {{ t('appName') }}</div>
      <h1>{{ metadata.title }}</h1>
      <p>{{ metadata.desc }}</p>
    </div>
    <div class="header-stat">
      <span class="stat-value">{{ store.files.length }}</span
      ><span class="stat-label">{{ t('files') }}</span>
    </div>
  </section>
  <section
    class="drop-zone"
    role="region"
    :aria-label="t('dropTitle')"
    :class="{ dragging }"
    @dragenter.prevent="onDragEnter"
    @dragover.prevent="dragging = true"
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
  >
    <div
      class="drop-main-action"
      role="button"
      tabindex="0"
      :aria-label="t('browse')"
      @click="browse"
      @keydown.enter.prevent="browse"
      @keydown.space.prevent="browse"
    >
      <div class="upload-icon">
        <Upload :size="22" />
      </div>
      <h2>{{ t('dropTitle') }}</h2>
      <p>{{ t('dropHint') }}</p>
    </div>
    <div class="drop-actions">
      <button class="secondary-button" @click.stop="browse">
        <Plus :size="16" /> {{ t('browse') }}</button
      ><button class="secondary-button" @click.stop="browseFolder">
        <FolderTree :size="16" /> {{ t('folder') }}
      </button>
    </div>
    <input
      ref="input"
      class="hidden-input"
      type="file"
      multiple
      :accept="
        store.activeTool === 'pdf'
          ? '.pdf,application/pdf'
          : '.png,.jpg,.jpeg,.webp,.avif,.ico,.svg,image/*'
      "
      @change="onInputChange"
    />
  </section>
  <div class="workspace-grid">
    <section class="queue-panel panel">
      <div class="panel-header">
        <div>
          <h2>{{ t('queue') }}</h2>
          <span class="muted">{{ store.files.length }} {{ t('files') }}</span>
        </div>
        <div class="queue-header-actions">
          <button
            v-if="store.files.some((item) => item.status === 'error')"
            class="text-button"
            :disabled="store.running"
            @click="store.retryFailed"
          >
            <RotateCcw :size="14" /> {{ t('retryFailed') }}</button
          ><button
            v-if="store.files.some((item) => item.status === 'done')"
            class="text-button"
            :disabled="store.running"
            @click="store.clearCompleted"
          >
            <Trash2 :size="14" /> {{ t('clearCompleted') }}</button
          ><button
            v-if="store.files.length"
            class="text-button"
            :disabled="store.running"
            @click="store.clearFiles"
          >
            <Trash2 :size="15" /> {{ t('clear') }}
          </button>
        </div>
      </div>
      <div
        v-if="store.files.length"
        class="queue-filters"
        role="group"
        :aria-label="t('queueFilter')"
      >
        <button
          v-for="filter in queueFilters"
          :key="filter.value"
          class="queue-filter"
          :class="{ selected: store.queueFilter === filter.value }"
          :aria-pressed="store.queueFilter === filter.value"
          @click="store.queueFilter = filter.value"
        >
          <span>{{ t(filter.label) }}</span
          ><span class="queue-filter-count">{{ filterCount(filter.value) }}</span>
        </button>
      </div>
      <div v-if="!store.files.length" class="empty-state">
        <div class="empty-icon">
          <Upload :size="20" />
        </div>
        <strong>{{ t('noFiles') }}</strong
        ><span>{{ t('noFilesHint') }}</span>
      </div>
      <div v-else-if="!store.visibleFiles.length" class="empty-state filtered-empty">
        <div class="empty-icon">
          <AlertCircle :size="20" />
        </div>
        <strong>{{ t('noMatches') }}</strong
        ><span>{{ t('noMatchesHint') }}</span>
      </div>
      <div v-else class="file-list">
        <div v-for="item in store.visibleFiles" :key="item.id" class="file-row">
          <div class="file-type">
            <img v-if="item.previewUrl" :src="item.previewUrl" :alt="item.name" />
            <component v-else :is="rowIcon(item.type)" :size="18" />
          </div>
          <div class="file-meta">
            <div class="file-name" :title="item.name">{{ item.name }}</div>
            <div class="file-sub">
              {{ formatSize(item.size)
              }}<span v-if="item.status === 'done'"> · {{ item.resultName }}</span>
            </div>
            <div
              v-if="item.size >= LARGE_IMAGE_BYTES && item.status === 'queued'"
              class="warning-copy"
            >
              <AlertCircle :size="13" /> {{ t('largeImageWarning') }}
            </div>
            <div
              v-if="
                store.activeTool === 'compress' &&
                (item.previewUrl || item.resultPreviewUrl || store.estimateForFile(item) > 0)
              "
              class="preview-strip"
            >
              <div v-if="item.previewUrl" class="preview-source">
                <img :src="item.previewUrl" :alt="item.name" />
              </div>
              <span v-else class="preview-placeholder"></span>
              <ArrowRight :size="13" />
              <div class="preview-target">
                <img
                  v-if="item.resultPreviewUrl"
                  :src="item.resultPreviewUrl"
                  :alt="item.resultName || item.name"
                /><span v-else class="preview-placeholder"></span
                ><span
                  v-if="item.status === 'done' && item.originalBytes && item.compressedBytes"
                  class="preview-estimate actual-size"
                  >{{ compressionComparison(item) }}</span
                ><span v-else-if="store.estimateForFile(item) > 0" class="preview-estimate"
                  >{{ t('estimatedShort') }} {{ formatSize(store.estimateForFile(item)) }}</span
                >
              </div>
            </div>
            <div
              v-if="item.status === 'processing' || item.status === 'queued'"
              class="progress-track"
              role="progressbar"
              :aria-valuenow="item.progress"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-label="`${item.name}: ${item.progress}%`"
            >
              <span :style="{ width: `${item.progress}%` }"></span>
            </div>
            <div v-if="item.status === 'error'" class="error-copy">
              <AlertCircle :size="13" /> {{ item.error || t('failed') }}
            </div>
            <div v-if="item.status === 'cancelled'" class="cancel-copy">
              <CircleSlash :size="13" /> {{ t('cancelled') }}
            </div>
            <div v-if="item.warning" class="warning-copy">{{ item.warning }}</div>
          </div>
          <div class="file-status">
            <Loader2 v-if="item.status === 'processing'" class="spin" :size="16" />
            <CheckCircle2 v-else-if="item.status === 'done'" class="success" :size="17" /><button
              v-else-if="item.status === 'error' || item.status === 'cancelled'"
              class="icon-button small"
              :disabled="store.running"
              :title="t('retry')"
              :aria-label="`${t('retry')}: ${item.name}`"
              @click="store.retry(item.id)"
            >
              <RotateCcw :size="15" />
            </button>
            <span v-else class="queue-dot"></span
            ><button
              class="icon-button small remove-button"
              :disabled="store.running || item.status === 'processing'"
              :title="t('remove')"
              :aria-label="`${t('remove')}: ${item.name}`"
              @click="store.removeFile(item.id)"
            >
              <X :size="15" />
            </button>
          </div>
        </div>
      </div>
    </section>
    <aside class="settings-panel panel">
      <h2>{{ t('preferences') }}</h2>
      <div class="field">
        <label for="output-directory">{{ t('output') }}</label>
        <div class="directory-input">
          <FolderOpen :size="16" /><input
            id="output-directory"
            v-model="store.outputDir"
            :placeholder="t('outputPlaceholder')"
            @change="store.setOutputDir(store.outputDir)"
          /><button
            class="directory-open-button"
            :title="t('openOutput')"
            :aria-label="t('openOutput')"
            @click="store.openOutputDirectory"
          >
            <FolderOpen :size="14" /></button
          ><button :title="t('change')" :aria-label="t('change')" @click="store.chooseOutput">
            <ChevronDown :size="15" />
          </button>
        </div>
      </div>
      <div v-if="store.activeTool === 'convert'" class="field">
        <label>{{ t('format') }}</label>
        <div class="segmented" role="group" :aria-label="t('format')">
          <button
            v-for="option in formatOptions"
            :key="option"
            :class="{ selected: format === option }"
            :aria-pressed="format === option"
            @click="format = option"
          >
            {{ option.toUpperCase() }}
          </button>
        </div>
      </div>
      <div v-if="store.activeTool === 'compress'" class="field">
        <div class="label-with-value">
          <label for="quality-input">{{ t('quality') }}</label
          ><strong>{{ quality }}%</strong>
        </div>
        <input id="quality-input" v-model.number="quality" class="range" type="range" min="10" max="100" step="1" />
        <div class="target-size-field">
          <label for="target-size-value">{{ t('targetBytes') }}</label>
          <div class="target-size-control">
            <input
              id="target-size-value"
              v-model.number="store.targetValue"
              class="target-size-input"
              type="number"
              min="0"
              :step="store.targetUnit === 'B' ? 1 : 0.01"
              inputmode="decimal"
              placeholder="0"
            /><select
              v-model="store.targetUnit"
              class="target-size-unit"
              :aria-label="t('targetBytes')"
            >
              <option v-for="unit in store.targetUnitOptions" :key="unit" :value="unit">
                {{ unit }}
              </option>
            </select>
          </div>
        </div>
        <label class="checkbox-row"
          ><input v-model="store.settings.lossless" type="checkbox" /><span>{{
            t('lossless')
          }}</span></label
        >
      </div>
      <div v-if="store.activeTool === 'pdf'" class="field">
        <div class="label-with-value">
          <label>{{ t('dpi') }}</label
          ><strong>{{ dpi }}</strong>
        </div>
        <div class="segmented dpi-segmented" role="group" :aria-label="t('dpi')">
          <button
            v-for="option in dpiOptions"
            :key="option"
            :class="{ selected: dpi === option }"
            :aria-pressed="dpi === option"
            @click="dpi = option"
          >
            {{ option }}
          </button>
        </div>
        <input
          v-model="store.settings.pageRange"
          class="text-input"
          :placeholder="t('pageRange')"
        />
      </div>
      <div class="field option-field">
        <label class="checkbox-row"
          ><input v-model="store.settings.recursive" type="checkbox" /><span>{{
            t('includeSubfolders')
          }}</span></label
        ><label v-if="store.activeTool !== 'pdf'" class="checkbox-row"
          ><input v-model="store.settings.preserveMetadata" type="checkbox" /><span>{{
            t('preserveMetadata')
          }}</span></label
        >
      </div>
      <div class="estimate">
        <span>{{
          store.activeTool === 'compress' ? t('estimatedCompressedSize') : t('estimate')
        }}</span
        ><strong>{{
          store.files.length
            ? store.activeTool === 'compress'
              ? `~${formatSize(store.estimatedTotalSize)}`
              : `${store.progress}%`
            : '—'
        }}</strong>
      </div>
    </aside>
  </div>
  <div
    v-if="store.files.length"
    class="sticky-action-bar"
    role="region"
    :aria-label="t('processingActions')"
  >
    <div class="sticky-summary">
      <strong>{{ store.files.length }}</strong> {{ t('files')
      }}<span>
        ·
        {{
          store.activeTool === 'compress'
            ? `~${formatSize(store.estimatedTotalSize)}`
            : `${store.progress}%`
        }}</span
      >
    </div>
    <div class="action-row">
      <button v-if="store.running" class="secondary-button" @click="store.cancel">
        <Square :size="15" /> {{ t('cancel') }}</button
      ><Button class="primary-button" :disabled="!store.canProcess" @click="store.process">
        <Loader2 v-if="store.running" class="spin" :size="17" />
        <Upload v-else :size="17" />{{ store.running ? t('processing') : t('start') }}
      </Button>
    </div>
  </div>
  <div
    v-if="
      store.completeCount &&
      !store.running &&
      !store.files.some((item) => item.status === 'queued' || item.status === 'processing')
    "
    class="toast"
  >
    <CheckCircle2 :size="17" /> {{ t('allDone') }} · {{ store.completeCount }} {{ t('files')
    }}<button
      class="icon-button small"
      :title="t('openFolder')"
      :aria-label="t('openFolder')"
      @click="store.openOutputDirectory"
    >
      <FolderOpen :size="15" />
    </button>
  </div>
</template>
