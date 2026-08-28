<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { AlertCircle, ArrowRight, CheckCircle2, ChevronDown, CircleSlash, FileImage, FileText, FolderOpen, Loader2, Plus, RotateCcw, Trash2, Upload, X, FolderTree, Square } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useWorkspaceStore } from '../stores/workspace'
import { wailsService } from '../services/wails'
import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import Button from '../components/ui/Button.vue'

const { t } = useI18n(); const store = useWorkspaceStore(); const input = ref<HTMLInputElement>(); const dragging = ref(false); let dragDepth = 0
const metadata = computed(() => store.activeTool === 'convert' ? { title: t('convert'), desc: t('convertDesc'), accepts: t('imageTypes'), icon: FileImage } : store.activeTool === 'compress' ? { title: t('compress'), desc: t('compressDesc'), accepts: t('imageTypes'), icon: FileImage } : { title: t('pdf'), desc: t('pdfDesc'), accepts: t('pdfType'), icon: FileText })
const formatOptions = ['webp', 'jpg', 'png', 'avif']
const dpiOptions = [72, 150, 300, 600]
const format = computed({ get: () => store.settings.format, set: (value: string) => { store.settings.format = value } })
const quality = computed({ get: () => store.settings.quality, set: (value: number) => { store.settings.quality = value } })
const dpi = computed({ get: () => store.settings.dpi, set: (value: number) => { store.settings.dpi = value } })
function chooseFiles(files: FileList | File[]) { store.addFiles(files); dragging.value = false; dragDepth = 0 }
function onDragEnter() { dragDepth += 1; dragging.value = true }
function onDragLeave() { dragDepth = Math.max(0, dragDepth - 1); if (dragDepth === 0) dragging.value = false }
function onDrop(event: DragEvent) { if (event.dataTransfer?.files) chooseFiles(event.dataTransfer.files); else { dragging.value = false; dragDepth = 0 } }
function onInputChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files) chooseFiles(target.files)
  target.value = ''
}
function browse() { if (wailsService.isNative()) void store.browseFiles(); else input.value?.click() }
function browseFolder() { void store.browseFolder() }
function formatSize(bytes: number) { if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`; return `${(bytes / 1024 / 1024).toFixed(1)} MB` }
function rowIcon(type: string) { return type === 'application/pdf' ? FileText : FileImage }
onMounted(() => { if (wailsService.isNative()) OnFileDrop((_x, _y, paths) => { void store.addNativePaths(paths) }, true) })
onUnmounted(() => { if (wailsService.isNative()) OnFileDropOff() })
</script>

<template>
  <section class="page-header"><div><div class="eyebrow"><component :is="metadata.icon" :size="14" /> {{ t('appName') }}</div><h1>{{ metadata.title }}</h1><p>{{ metadata.desc }}</p></div><div class="header-stat"><span class="stat-value">{{ store.files.length }}</span><span class="stat-label">{{ t('files') }}</span></div></section>
  <section class="drop-zone" :class="{ dragging }" @dragenter.prevent="onDragEnter" @dragover.prevent="dragging = true" @dragleave.prevent="onDragLeave" @drop.prevent="onDrop" @click="browse">
    <div class="upload-icon"><Upload :size="22" /></div><h2>{{ t('dropTitle') }}</h2><p>{{ t('dropHint') }}</p><div class="drop-actions"><button class="secondary-button" @click.stop="browse"><Plus :size="16" /> {{ t('browse') }}</button><button class="secondary-button" @click.stop="browseFolder"><FolderTree :size="16" /> {{ t('folder') }}</button></div><input ref="input" class="hidden-input" type="file" multiple :accept="store.activeTool === 'pdf' ? '.pdf,application/pdf' : 'image/*'" @change="onInputChange" />
  </section>
  <div class="workspace-grid">
    <section class="queue-panel panel"><div class="panel-header"><div><h2>{{ t('queue') }}</h2><span class="muted">{{ store.files.length }} {{ t('files') }}</span></div><button v-if="store.files.length" class="text-button" :disabled="store.running" @click="store.clearFiles"><Trash2 :size="15" /> {{ t('clear') }}</button></div>
      <div v-if="!store.files.length" class="empty-state"><div class="empty-icon"><Upload :size="20" /></div><strong>{{ t('noFiles') }}</strong><span>{{ t('noFilesHint') }}</span></div>
      <div v-else class="file-list"><div v-for="item in store.files" :key="item.id" class="file-row"><div class="file-type"><img v-if="item.previewUrl" :src="item.previewUrl" :alt="item.name" /><component v-else :is="rowIcon(item.type)" :size="18" /></div><div class="file-meta"><div class="file-name" :title="item.name">{{ item.name }}</div><div class="file-sub">{{ formatSize(item.size) }}<span v-if="item.status === 'done'"> · {{ item.resultName }}</span></div><div v-if="store.activeTool === 'compress' && (item.previewUrl || item.resultPreviewUrl)" class="preview-strip"><img v-if="item.previewUrl" :src="item.previewUrl" :alt="item.name" /><ArrowRight :size="13" /><img v-if="item.resultPreviewUrl" :src="item.resultPreviewUrl" :alt="item.resultName || item.name" /><span v-else class="preview-placeholder"></span></div><div v-if="item.status === 'processing' || item.status === 'queued'" class="progress-track"><span :style="{ width: `${item.progress}%` }"></span></div><div v-if="item.status === 'error'" class="error-copy"><AlertCircle :size="13" /> {{ item.error || t('failed') }}</div><div v-if="item.status === 'cancelled'" class="cancel-copy"><CircleSlash :size="13" /> {{ t('cancelled') }}</div><div v-if="item.warning" class="warning-copy">{{ item.warning }}</div></div><div class="file-status"><Loader2 v-if="item.status === 'processing'" class="spin" :size="16" /><CheckCircle2 v-else-if="item.status === 'done'" class="success" :size="17" /><button v-else-if="item.status === 'error'" class="icon-button small" :title="t('retry')" @click="store.retry(item.id)"><RotateCcw :size="15" /></button><CircleSlash v-else-if="item.status === 'cancelled'" class="cancelled" :size="16" /><span v-else class="queue-dot"></span><button class="icon-button small remove-button" :disabled="item.status === 'processing'" :title="t('remove')" @click="store.removeFile(item.id)"><X :size="15" /></button></div></div></div>
    </section>
    <aside class="settings-panel panel"><h2>{{ t('preferences') }}</h2><div class="field"><label>{{ t('output') }}</label><div class="directory-input"><FolderOpen :size="16" /><input v-model="store.outputDir" :placeholder="t('outputPlaceholder')" @change="store.setOutputDir(store.outputDir)" /><button :title="t('change')" @click="store.chooseOutput"><ChevronDown :size="15" /></button></div></div><div v-if="store.activeTool === 'convert'" class="field"><label>{{ t('format') }}</label><div class="segmented"><button v-for="option in formatOptions" :key="option" :class="{ selected: format === option }" @click="format = option">{{ option.toUpperCase() }}</button></div></div><div v-if="store.activeTool === 'compress'" class="field"><div class="label-with-value"><label>{{ t('quality') }}</label><strong>{{ quality }}%</strong></div><input v-model.number="quality" class="range" type="range" min="10" max="100" step="1" /><label class="checkbox-row"><input v-model.number="store.settings.targetBytes" type="number" min="0" step="1024" placeholder="0" /><span>{{ t('targetBytes') }}</span></label><label class="checkbox-row"><input v-model="store.settings.lossless" type="checkbox" /><span>{{ t('lossless') }}</span></label></div><div v-if="store.activeTool === 'pdf'" class="field"><div class="label-with-value"><label>{{ t('dpi') }}</label><strong>{{ dpi }}</strong></div><div class="segmented dpi-segmented"><button v-for="option in dpiOptions" :key="option" :class="{ selected: dpi === option }" @click="dpi = option">{{ option }}</button></div><input v-model="store.settings.pageRange" class="text-input" :placeholder="t('pageRange')" /></div><div class="field option-field"><label class="checkbox-row"><input v-model="store.settings.recursive" type="checkbox" /><span>{{ t('includeSubfolders') }}</span></label><label v-if="store.activeTool !== 'pdf'" class="checkbox-row"><input v-model="store.settings.preserveMetadata" type="checkbox" /><span>{{ t('preserveMetadata') }}</span></label></div><div class="estimate"><span>{{ t('estimate') }}</span><strong>{{ store.files.length ? `${store.progress}%` : '—' }}</strong></div><div class="action-row"><button v-if="store.running" class="secondary-button" @click="store.cancel"><Square :size="15" /> {{ t('cancel') }}</button><Button class="primary-button" :disabled="!store.canProcess" @click="store.process"><Loader2 v-if="store.running" class="spin" :size="17" /><Upload v-else :size="17" />{{ store.running ? t('processing') : t('start') }}</Button></div></aside>
  </div>
  <div v-if="store.completeCount && !store.running && !store.files.some(item => item.status === 'queued' || item.status === 'processing')" class="toast"><CheckCircle2 :size="17" /> {{ t('allDone') }} · {{ store.completeCount }} {{ t('files') }}<button class="icon-button small" :title="t('openFolder')" @click="store.openOutputDirectory"><FolderOpen :size="15" /></button></div>
</template>
