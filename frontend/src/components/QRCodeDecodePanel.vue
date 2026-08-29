<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { AlertCircle, CheckCircle2, Copy, FileImage, Loader2, Plus, QrCode, Upload, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { ClipboardSetText, OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import Button from './ui/Button.vue'
import { wailsService } from '../services/wails'
import { QR_CODE_MAX_DECODE_PIXELS } from '../qrcode'
import type { NativeInputFile } from '../types'

const { t } = useI18n()
const input = ref<HTMLInputElement>()
const dragging = ref(false)
const source = ref<DecodeSource>()
const previewUrl = ref('')
const decodedText = ref('')
const loading = ref(false)
const error = ref('')
const copied = ref(false)
let requestId = 0
let selectionId = 0
let objectUrl = ''
let copyTimer: number | undefined
let mounted = false

interface DecodeSource {
  path?: string
  file?: File
  name: string
  size: number
}

const canCopy = computed(() => Boolean(decodedText.value) && !loading.value)
const sourceSize = computed(() => {
  const bytes = source.value?.size ?? 0
  if (bytes < 1024) return `${Math.max(0, bytes)} B`
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
})

function isImageFile(file: File) {
  const extension = file.name.match(/\.[^/.]+$/)?.[0].toLowerCase() ?? ''
  return file.type.startsWith('image/') || ['.png', '.jpg', '.jpeg', '.webp', '.avif', '.ico', '.svg'].includes(extension)
}

function releaseObjectUrl() {
  if (!objectUrl) return
  URL.revokeObjectURL(objectUrl)
  objectUrl = ''
}

function resetCopiedState() {
  if (copyTimer !== undefined) {
    window.clearTimeout(copyTimer)
    copyTimer = undefined
  }
  copied.value = false
}

function clearSelection() {
  requestId += 1
  selectionId += 1
  releaseObjectUrl()
  source.value = undefined
  previewUrl.value = ''
  decodedText.value = ''
  error.value = ''
  loading.value = false
  resetCopiedState()
}

function displayError(reason: unknown) {
  const message = reason instanceof Error ? reason.message : String(reason ?? '')
  if (/unavailable|desktop app|environment|browser/i.test(message)) return t('qrDecodeUnavailable')
  if (/no qr|not found|没有二维码/i.test(message)) return t('qrDecodeNoCode')
  return t('qrDecodeFailed')
}

async function decode(next: DecodeSource) {
  if (!mounted) return
  const currentRequest = ++requestId
  loading.value = true
  error.value = ''
  decodedText.value = ''
  resetCopiedState()
  source.value = next
  previewUrl.value = ''
  releaseObjectUrl()

  try {
    let imageUrl = ''
    let decodePath = next.path ?? ''
    if (next.file) {
      objectUrl = URL.createObjectURL(next.file)
      imageUrl = objectUrl
      decodePath = objectUrl
    } else if (next.path) {
      try {
        imageUrl = (await wailsService.previewImage(next.path, { maxDimension: 960, maxPixels: QR_CODE_MAX_DECODE_PIXELS })).dataUrl ?? ''
      } catch {
        // A missing thumbnail should not prevent the native decoder from
        // inspecting the original path.
      }
    }
    if (currentRequest === requestId) previewUrl.value = imageUrl
    if (currentRequest !== requestId) return
    const result = await wailsService.decodeQRCode(decodePath)
    if (currentRequest !== requestId) return
    previewUrl.value = imageUrl
    decodedText.value = result.text
  } catch (reason) {
    if (currentRequest !== requestId) return
    error.value = displayError(reason)
  } finally {
    if (currentRequest === requestId) loading.value = false
  }
}

function chooseBrowserFiles(files: FileList | File[]) {
  if (!files.length) return
  const file = Array.from(files).find(isImageFile)
  dragging.value = false
  if (!file) {
    clearSelection()
    error.value = t('qrDecodeImageOnly')
    return
  }
  void decode({ file, name: file.name, size: file.size })
}

async function chooseNativeFiles(files?: NativeInputFile[], requestedSelectionId?: number) {
  const currentSelection = requestedSelectionId ?? ++selectionId
  const fromDialog = files === undefined
  try {
    const selected = files ?? await wailsService.openInputFiles()
    if (!mounted || currentSelection !== selectionId) return
    if (!selected.length) {
      if (!fromDialog) {
        clearSelection()
        error.value = t('qrDecodeImageOnly')
      }
      return
    }
    const file = selected.find(item => item.kind === 'image')
    if (!file) {
      clearSelection()
      error.value = t('qrDecodeImageOnly')
      return
    }
    void decode({ path: file.path, name: file.name, size: file.size })
  } catch {
    // Cancelling the native dialog is intentionally a no-op.
  }
}

function browse() {
  if (wailsService.isNative()) void chooseNativeFiles()
  else input.value?.click()
}

function onInputChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files?.length) chooseBrowserFiles(target.files)
  target.value = ''
}

function onDragEnter() { dragging.value = true }
function onDragLeave() { dragging.value = false }
function onDrop(event: DragEvent) {
  dragging.value = false
  if (wailsService.isNative()) return
  if (event.dataTransfer?.files?.length) chooseBrowserFiles(event.dataTransfer.files)
}

async function copyDecodedText() {
  if (!canCopy.value) return
  const text = decodedText.value
  const currentRequest = requestId
  try {
    if (wailsService.isNative()) await ClipboardSetText(text)
    else await navigator.clipboard.writeText(text)
    if (!mounted || currentRequest !== requestId || decodedText.value !== text) return
    copied.value = true
    if (copyTimer !== undefined) window.clearTimeout(copyTimer)
    copyTimer = window.setTimeout(() => {
      if (mounted && currentRequest === requestId && decodedText.value === text) copied.value = false
      copyTimer = undefined
    }, 1600)
  } catch {
    // Clipboard permissions are optional; the decoded text remains selectable.
  }
}

onMounted(() => {
  mounted = true
  if (!wailsService.isNative()) return
  OnFileDrop((_x, _y, paths) => {
    const currentSelection = ++selectionId
    void wailsService.openInputFilesFromPaths(paths).then(files => {
      if (mounted && currentSelection === selectionId) return chooseNativeFiles(files, currentSelection)
      return undefined
    }).catch(() => undefined)
  }, true)
})

onUnmounted(() => {
  mounted = false
  requestId += 1
  selectionId += 1
  releaseObjectUrl()
  resetCopiedState()
  if (wailsService.isNative()) OnFileDropOff()
})
</script>

<template>
  <div class="qr-decode-workspace">
    <section class="panel qr-decode-main-panel">
      <div class="panel-header qr-decode-panel-header">
        <div><h2>{{ t('qrDecodePreview') }}</h2><span class="muted" :title="source?.name">{{ source?.name || t('qrDecodeEmpty') }}</span></div>
        <span v-if="source" class="qr-format-badge">{{ sourceSize }}</span>
      </div>
      <div class="qr-decode-preview-body">
        <div class="qr-decode-preview-surface" :class="{ empty: !previewUrl }">
          <img v-if="previewUrl" :src="previewUrl" :alt="source?.name || t('qrDecodePreview')" />
          <div v-if="loading" class="qr-preview-state" role="status"><Loader2 class="spin" :size="24" /><span>{{ t('qrDecoding') }}</span></div>
          <div v-else-if="error" class="qr-preview-state error" role="status"><AlertCircle :size="24" /><span>{{ error }}</span></div>
          <div v-else-if="!previewUrl" class="qr-preview-state"><QrCode :size="28" /><span>{{ t('qrDecodeEmpty') }}</span></div>
        </div>
      </div>
      <div v-if="decodedText" class="qr-decoded-result">
        <div class="qr-decoded-result-heading"><div><span class="qr-result-kicker"><CheckCircle2 :size="14" /> {{ t('qrDecodedText') }}</span><span class="muted">{{ decodedText.length }} {{ t('qrContentLength') }}</span></div><button class="icon-button small" :title="copied ? t('qrCopied') : t('qrCopyText')" :aria-label="copied ? t('qrCopied') : t('qrCopyText')" @click="copyDecodedText"><CheckCircle2 v-if="copied" :size="15" class="success" /><Copy v-else :size="15" /></button></div>
        <textarea class="qr-decoded-text" readonly :value="decodedText" :aria-label="t('qrDecodedText')"></textarea>
      </div>
    </section>

    <aside class="settings-panel panel qr-decode-settings-panel">
      <h2>{{ t('qrDecodeTab') }}</h2>
      <section class="qr-decode-drop-zone" :class="{ dragging }" @dragenter.prevent="onDragEnter" @dragover.prevent="dragging = true" @dragleave.prevent="onDragLeave" @drop.prevent="onDrop" @click="browse">
        <div class="upload-icon"><Upload :size="21" /></div>
        <h3>{{ t('qrDecodeDropTitle') }}</h3>
        <p>{{ t('qrDecodeDropHint') }}</p>
        <Button class="secondary-button" @click.stop="browse"><Plus :size="16" /> {{ t('qrDecodeBrowse') }}</Button>
        <input ref="input" class="hidden-input" type="file" accept=".png,.jpg,.jpeg,.webp,.avif,.ico,.svg,image/*" @change="onInputChange" />
      </section>
      <div v-if="source" class="qr-selected-file"><FileImage :size="17" /><span :title="source.name">{{ source.name }}</span><button class="icon-button small" :title="t('clear')" :aria-label="t('clear')" @click="clearSelection"><X :size="14" /></button></div>
      <div class="qr-decode-note">{{ t('qrDecodeDesc') }}</div>
    </aside>
  </div>
</template>
