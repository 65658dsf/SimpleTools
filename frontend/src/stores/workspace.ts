import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import type { JobItem, JobRequest, JobStatus, NativeInputFile, QueueFile, ToolId } from '../types'
import { wailsService } from '../services/wails'

function id() { return globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random()}` }
function stripExtension(name: string) { return name.replace(/\.[^/.]+$/, '') }
function extension(name: string) { return name.match(/\.[^/.]+$/)?.[0] || '.jpg' }
function mimeFor(name: string) {
  const ext = extension(name).toLowerCase()
  if (ext === '.pdf') return 'application/pdf'
  if (ext === '.jpg' || ext === '.jpeg') return 'image/jpeg'
  if (ext === '.ico') return 'image/x-icon'
  if (ext === '.svg') return 'image/svg+xml'
  return `image/${ext.slice(1)}`
}

const FORMAT_OPTIONS = ['webp', 'jpg', 'png', 'avif', 'ico', 'svg'] as const
const DPI_OPTIONS = [72, 150, 300, 600] as const
export const TARGET_BYTES_UNITS = ['B', 'KB', 'MB', 'GB'] as const
export type TargetBytesUnit = typeof TARGET_BYTES_UNITS[number]

const BYTES_PER_TARGET_UNIT: Record<TargetBytesUnit, number> = {
  B: 1,
  KB: 1024,
  MB: 1024 * 1024,
  GB: 1024 * 1024 * 1024,
}

const MAX_TARGET_BYTES = Number.MAX_SAFE_INTEGER

type WorkspaceSettings = {
  format: string
  quality: number
  targetBytes: number
  targetBytesUnit: TargetBytesUnit
  targetBytesUnitAuto: boolean
  dpi: number
  pageRange: string
  recursive: boolean
  preserveMetadata: boolean
  lossless: boolean
}

const DEFAULT_SETTINGS: WorkspaceSettings = {
  format: 'webp',
  quality: 76,
  targetBytes: 0,
  targetBytesUnit: 'KB',
  targetBytesUnitAuto: true,
  dpi: 150,
  pageRange: '',
  recursive: true,
  preserveMetadata: false,
  lossless: false,
}

function normalizeTargetUnit(value: unknown, fallback = DEFAULT_SETTINGS.targetBytesUnit): TargetBytesUnit {
  return typeof value === 'string' && TARGET_BYTES_UNITS.includes(value as TargetBytesUnit)
    ? value as TargetBytesUnit
    : fallback
}

/** Pick a human-friendly binary unit for an input byte count. */
export function targetUnitForBytes(bytes: number): TargetBytesUnit {
  const normalized = Number.isFinite(bytes) ? Math.max(0, bytes) : 0
  if (normalized >= BYTES_PER_TARGET_UNIT.GB) return 'GB'
  if (normalized >= BYTES_PER_TARGET_UNIT.MB) return 'MB'
  if (normalized >= BYTES_PER_TARGET_UNIT.KB) return 'KB'
  return 'B'
}

export function targetBytesToValue(bytes: number, unit: TargetBytesUnit): number {
  const normalized = Number.isFinite(bytes) ? Math.max(0, bytes) : 0
  if (normalized === 0) return 0
  const value = normalized / BYTES_PER_TARGET_UNIT[unit]
  return unit === 'B' ? Math.round(value) : Number(value.toFixed(3))
}

export function targetValueToBytes(value: number, unit: TargetBytesUnit): number {
  const normalized = Number.isFinite(value) ? Math.max(0, value) : 0
  if (normalized === 0) return 0
  return Math.min(MAX_TARGET_BYTES, Math.max(0, Math.round(normalized * BYTES_PER_TARGET_UNIT[unit])))
}

/**
 * Return a deliberately conservative client-side estimate for a compressed
 * image. The real encoder remains the source of truth; this helper only uses
 * the input byte count and current options so no image data crosses IPC.
 */
export function estimateCompressedSize(originalBytes: number, quality = 76, targetBytes = 0, lossless = false, mimeType = ''): number {
  const original = Number.isFinite(originalBytes) ? Math.max(0, Math.round(originalBytes)) : 0
  if (original === 0) return 0

  const target = Number.isFinite(targetBytes) ? Math.max(0, Math.round(targetBytes)) : 0
  const normalizedQuality = Number.isFinite(quality) ? Math.min(100, Math.max(10, quality)) : 76

  // PNG and the explicit lossless mode do not respond to a quality slider.
  // A small reduction is typical after metadata removal, but the result can
  // still be larger for some images, so never promise more than an estimate.
  const normalizedMime = mimeType.toLowerCase()
  if (lossless || normalizedMime === 'image/png' || normalizedMime === 'image/x-icon' || normalizedMime === 'image/vnd.microsoft.icon' || normalizedMime === 'image/svg+xml') {
    return Math.max(1, Math.round(original * 0.92))
  }

  // A linear approximation is easier to understand and remains monotonic as
  // the user moves the quality slider. The output is marked as approximate in
  // the UI because image content can move the actual size substantially.
  const qualityRatio = 0.25 + (normalizedQuality / 100) * 0.75
  const qualityEstimate = Math.max(1, Math.round(original * qualityRatio))
  if (target > 0) return Math.max(1, Math.min(qualityEstimate, target))
  return qualityEstimate
}

function readStorage(key: string): string | null {
  if (typeof localStorage === 'undefined') return null
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function writeStorage(key: string, value: string) {
  if (typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(key, value)
  } catch {
    // Preferences are best effort and must not block file processing.
  }
}

function readSettings(): WorkspaceSettings {
  const raw = readStorage('simpletools-settings')
  if (!raw) return { ...DEFAULT_SETTINGS }
  try {
    const saved = JSON.parse(raw) as Partial<WorkspaceSettings>
    const format = typeof saved.format === 'string' && FORMAT_OPTIONS.includes(saved.format as typeof FORMAT_OPTIONS[number]) ? saved.format : DEFAULT_SETTINGS.format
    const quality = typeof saved.quality === 'number' && Number.isFinite(saved.quality) ? Math.min(100, Math.max(10, Math.round(saved.quality))) : DEFAULT_SETTINGS.quality
    const targetBytes = typeof saved.targetBytes === 'number' && Number.isFinite(saved.targetBytes) ? Math.max(0, Math.round(saved.targetBytes)) : DEFAULT_SETTINGS.targetBytes
    const targetBytesUnit = normalizeTargetUnit(saved.targetBytesUnit)
    const targetBytesUnitAuto = typeof saved.targetBytesUnitAuto === 'boolean' ? saved.targetBytesUnitAuto : targetBytes === 0
    const dpi = typeof saved.dpi === 'number' && DPI_OPTIONS.includes(saved.dpi as typeof DPI_OPTIONS[number]) ? saved.dpi : DEFAULT_SETTINGS.dpi
    return {
      format,
      quality,
      targetBytes,
      targetBytesUnit,
      targetBytesUnitAuto,
      dpi,
      pageRange: typeof saved.pageRange === 'string' ? saved.pageRange : DEFAULT_SETTINGS.pageRange,
      recursive: typeof saved.recursive === 'boolean' ? saved.recursive : DEFAULT_SETTINGS.recursive,
      preserveMetadata: typeof saved.preserveMetadata === 'boolean' ? saved.preserveMetadata : DEFAULT_SETTINGS.preserveMetadata,
      lossless: typeof saved.lossless === 'boolean' ? saved.lossless : DEFAULT_SETTINGS.lossless,
    }
  } catch {
    return { ...DEFAULT_SETTINGS }
  }
}

export const useWorkspaceStore = defineStore('workspace', () => {
  const activeTool = ref<ToolId>('convert')
  const files = ref<QueueFile[]>([])
  const outputDir = ref(readStorage('simpletools-output') ?? '')
  const running = ref(false)
  const activeJobId = ref<string>()
  const settings = ref<WorkspaceSettings>(readSettings())
  const cancelRequested = ref(false)
  let stopProgress: () => void = () => undefined
  let stopItems: () => void = () => undefined

  const completeCount = computed(() => files.value.filter(item => item.status === 'done').length)
  const progress = computed(() => files.value.length ? Math.round(files.value.reduce((sum, item) => sum + item.progress, 0) / files.value.length) : 0)
  const targetUnit = computed<TargetBytesUnit>({
    get: () => settings.value.targetBytesUnit,
    set: value => {
      const normalized = normalizeTargetUnit(value)
      settings.value.targetBytesUnit = normalized
      settings.value.targetBytesUnitAuto = false
    },
  })
  const targetValue = computed<number>({
    get: () => targetBytesToValue(settings.value.targetBytes, settings.value.targetBytesUnit),
    set: value => {
      const numeric = typeof value === 'number' ? value : Number(value)
      settings.value.targetBytes = targetValueToBytes(numeric, settings.value.targetBytesUnit)
      settings.value.targetBytesUnitAuto = false
    },
  })
  const estimatedTotalSize = computed(() => activeTool.value === 'compress'
    ? files.value.reduce((sum, item) => sum + estimateCompressedSize(item.size, settings.value.quality, settings.value.targetBytes, settings.value.lossless, item.type), 0)
    : 0)
  const isEligible = (item: QueueFile) => settings.value.recursive || !item.relativePath
  const canProcess = computed(() => {
    return !running.value && files.value.some(item => isEligible(item) && (item.status === 'queued' || item.status === 'error'))
  })

  function autoSelectTargetUnit() {
    if (!settings.value.targetBytesUnitAuto || activeTool.value !== 'compress') return
    const largestInput = files.value.reduce((largest, item) => Math.max(largest, item.size), 0)
    if (largestInput <= 0) return
    settings.value.targetBytesUnit = targetUnitForBytes(largestInput)
  }

  watch(outputDir, value => writeStorage('simpletools-output', value))
  watch(settings, value => writeStorage('simpletools-settings', JSON.stringify(value)), { deep: true })

  function acceptsNative(file: NativeInputFile) {
    if (activeTool.value === 'pdf' ? file.kind !== 'pdf' : file.kind !== 'image') return false
    return settings.value.recursive || !file.relativePath
  }
  function acceptsBrowser(file: File) {
    const ext = file.name.match(/\.[^/.]+$/)?.[0].toLowerCase()
    if (activeTool.value === 'pdf') return file.type === 'application/pdf' || ext === '.pdf'
    const supportedExtension = ['.png', '.jpg', '.jpeg', '.webp', '.avif', '.ico', '.svg'].includes(ext ?? '')
    return supportedExtension && (!file.type || file.type.startsWith('image/') || ext === '.ico' || ext === '.svg')
  }

  function setTool(tool: ToolId) {
    if (tool !== 'convert' && tool !== 'compress' && tool !== 'pdf') return
    activeTool.value = tool
    files.value = files.value.filter(item => activeTool.value === 'pdf' ? item.type === 'application/pdf' : item.type.startsWith('image/'))
    autoSelectTargetUnit()
  }

  function addNativeFiles(selected: NativeInputFile[]) {
    selected.filter(acceptsNative).forEach(file => {
      // Two files in different folders may legitimately share a name and size.
      // Native paths are the stable identity; do not dedupe by display metadata.
      if (files.value.some(entry => entry.path === file.path)) return
      const item: QueueFile = { id: id(), path: file.path, relativePath: file.relativePath, name: file.name, size: file.size, type: mimeFor(file.name), status: 'queued', progress: 0 }
      files.value.push(item)
      void loadPreview(item)
    })
    autoSelectTargetUnit()
  }

  function addFiles(selected: FileList | File[]) {
    Array.from(selected).filter(acceptsBrowser).forEach(file => {
      if (files.value.some(entry => entry.file === file)) return
      const item: QueueFile = { id: id(), file, name: file.name, size: file.size, type: file.type || mimeFor(file.name), status: 'queued', progress: 0 }
      files.value.push(item)
      void loadPreview(item)
    })
    autoSelectTargetUnit()
  }

  async function loadPreview(item: QueueFile, target = false) {
    try {
      let url = ''
      if (item.file) url = await browserThumbnail(item.file)
      else if (item.path) url = (await wailsService.previewImage(item.path, { maxDimension: 240 })).dataUrl ?? ''
      if (target) item.resultPreviewUrl = url
      else item.previewUrl = url
    } catch {
      // A preview is optional and must never block processing.
    }
  }

  function releasePreview(item: QueueFile) {
    for (const url of [item.previewUrl, item.resultPreviewUrl]) {
      if (url?.startsWith('blob:')) URL.revokeObjectURL(url)
    }
  }

  function releaseResultPreview(item: QueueFile) {
    if (item.resultPreviewUrl?.startsWith('blob:')) URL.revokeObjectURL(item.resultPreviewUrl)
  }

  async function browserThumbnail(file: File, maxDimension = 240): Promise<string> {
    if (typeof document === 'undefined' || typeof URL === 'undefined' || typeof URL.createObjectURL !== 'function') return ''
    const sourceURL = URL.createObjectURL(file)
    try {
      const img = new Image()
      img.src = sourceURL
      await new Promise<void>((resolve, reject) => {
        img.onload = () => resolve()
        img.onerror = () => reject(new Error('Unable to preview image'))
      })
      const scale = Math.min(1, maxDimension / Math.max(img.naturalWidth || 1, img.naturalHeight || 1))
      const canvas = document.createElement('canvas')
      canvas.width = Math.max(1, Math.round((img.naturalWidth || 1) * scale))
      canvas.height = Math.max(1, Math.round((img.naturalHeight || 1) * scale))
      const context = canvas.getContext('2d')
      if (!context) return ''
      context.drawImage(img, 0, 0, canvas.width, canvas.height)
      return canvas.toDataURL('image/jpeg', 0.78)
    } finally {
      URL.revokeObjectURL(sourceURL)
    }
  }

  async function browseFiles() {
    if (!wailsService.isNative()) return
    addNativeFiles(await wailsService.openInputFiles())
  }

  async function browseFolder() {
    if (!wailsService.isNative()) return
    addNativeFiles(await wailsService.openInputFolder())
  }

  async function addNativePaths(paths: string[]) {
    if (!wailsService.isNative() || paths.length === 0) return
    addNativeFiles(await wailsService.openInputFilesFromPaths(paths))
  }

  async function chooseOutput() {
    if (!wailsService.isNative()) return
    const selected = await wailsService.chooseOutputDirectory()
    if (selected) setOutputDir(selected)
  }

  async function loadDefaultOutputDirectory() {
    if (!wailsService.isNative() || outputDir.value.trim()) return
    try {
      const selected = await wailsService.getDefaultOutputDirectory()
      if (selected && !outputDir.value.trim()) setOutputDir(selected)
    } catch {
      // The backend still applies the default when processing starts.
    }
  }

  async function openOutputDirectory() {
    if (!wailsService.isNative()) return
    await loadDefaultOutputDirectory()
    if (outputDir.value.trim()) await wailsService.openOutputDirectory(outputDir.value)
  }

  function removeFile(fileId: string) { const item = files.value.find(entry => entry.id === fileId); if (!item || item.status === 'processing') return; releasePreview(item); files.value = files.value.filter(entry => entry.id !== fileId); autoSelectTargetUnit() }
  function clearFiles() { if (running.value) return; files.value.forEach(releasePreview); files.value = []; autoSelectTargetUnit() }
  function retry(fileId: string) { const item = files.value.find(entry => entry.id === fileId); if (item) { releaseResultPreview(item); item.status = 'queued'; item.progress = 0; item.error = undefined; item.warning = undefined; item.resultName = undefined; item.resultNames = undefined; item.resultPreviewUrl = undefined } }
  function setOutputDir(dir: string) { outputDir.value = dir }
  function estimateForFile(item: QueueFile) {
    return estimateCompressedSize(item.size, settings.value.quality, settings.value.targetBytes, settings.value.lossless, item.type)
  }

  function nativeRequest(source = files.value.filter(item => item.status === 'queued' || item.status === 'error')): JobRequest {
    return {
      tool: activeTool.value,
      inputs: source.map(item => item.path).filter((path): path is string => Boolean(path)),
      inputRelativeDirs: Object.fromEntries(source.filter(item => item.path && item.relativePath).map(item => [item.path, item.relativePath])) as Record<string, string>,
      outputDirectory: outputDir.value,
      format: activeTool.value === 'convert' ? settings.value.format : undefined,
      quality: activeTool.value === 'compress' ? settings.value.quality : undefined,
      targetBytes: activeTool.value === 'compress' && settings.value.targetBytes > 0 ? settings.value.targetBytes : undefined,
      lossless: activeTool.value === 'compress' ? settings.value.lossless : undefined,
      dpi: activeTool.value === 'pdf' ? settings.value.dpi : undefined,
      pageRange: activeTool.value === 'pdf' ? settings.value.pageRange : undefined,
      recursive: settings.value.recursive,
      preserveMetadata: settings.value.preserveMetadata,
    }
  }

  async function process() {
    const processable = files.value.filter(item => isEligible(item) && (item.status === 'queued' || item.status === 'error'))
    if (!processable.length || running.value) return
    running.value = true
    cancelRequested.value = false
    processable.forEach(item => { item.status = 'processing'; item.progress = 0; item.error = undefined })
    if (wailsService.isNative() && processable.every(item => item.path)) {
      try {
        stopProgress = wailsService.onJobProgress(applyStatus)
        stopItems = wailsService.onJobItem(applyItem)
        activeJobId.value = await wailsService.startJob(nativeRequest(processable))
        if (cancelRequested.value) {
          try { await wailsService.cancelJob(activeJobId.value) } catch { /* status polling still determines the final state */ }
        }
        let status = await wailsService.getJob(activeJobId.value)
        while (status.state === 'queued' || status.state === 'running') {
          await new Promise(resolve => setTimeout(resolve, 150))
          status = await wailsService.getJob(activeJobId.value)
        }
        applyStatus(status)
      } catch (error) {
        files.value.forEach(item => { if (item.status === 'processing') { item.status = 'error'; item.error = error instanceof Error ? error.message : 'Processing failed' } })
      } finally {
        stopProgress(); stopItems(); stopProgress = () => undefined; stopItems = () => undefined; activeJobId.value = undefined; running.value = false
        cancelRequested.value = false
      }
      return
    }
    // Browser preview mode intentionally simulates work because paths are not available.
    for (const item of processable) {
      if (cancelRequested.value) {
        item.status = 'cancelled'
        continue
      }
      const completed = await simulateProgress(item)
      if (!completed) {
        item.status = 'cancelled'
        break
      }
      item.resultName = activeTool.value === 'pdf' ? `${stripExtension(item.name)}-page-001.png` : activeTool.value === 'compress' ? `${stripExtension(item.name)}-compressed${extension(item.name)}` : `${stripExtension(item.name)}.${settings.value.format}`
      item.progress = 100; item.status = 'done'
    }
    if (cancelRequested.value) processable.filter(item => item.status === 'processing').forEach(item => { item.status = 'cancelled' })
    running.value = false
    cancelRequested.value = false
  }

  async function cancel() {
    if (!running.value) return
    cancelRequested.value = true
    if (activeJobId.value) {
      try { await wailsService.cancelJob(activeJobId.value) } catch { /* the job will surface cancellation through its status event */ }
    }
  }

  function applyStatus(status: JobStatus) {
    status.items?.forEach(applyItem)
    files.value.forEach(item => {
      if (item.path && status.current === item.path && item.status === 'queued') item.status = 'processing'
    })
  }

  function applyItem(item: JobItem) {
    let target = files.value.find(entry => entry.path === item.path)
    if (!target) {
      const sameName = files.value.filter(entry => entry.name === item.name)
      if (sameName.length === 1) target = sameName[0]
    }
    if (!target) return
    target.progress = item.progress >= 1 ? 100 : Math.round(item.progress * 100)
    if (item.state === 'completed') target.status = 'done'
    else if (item.state === 'failed') target.status = 'error'
    else if (item.state === 'cancelled') target.status = 'cancelled'
    else target.status = 'processing'
    target.error = item.error
    target.warning = item.warning
    target.resultName = item.output?.split(/[\\/]/).pop()
    target.resultNames = item.outputs
    if (item.output && activeTool.value !== 'pdf') void loadPreview(target, true)
  }

  async function simulateProgress(item: QueueFile) { for (const value of [20, 45, 70, 90]) { await new Promise(resolve => setTimeout(resolve, 100)); if (cancelRequested.value) return false; item.progress = value } return true }

  return { activeTool, files, outputDir, running, settings, targetValue, targetUnit, targetUnitOptions: TARGET_BYTES_UNITS, completeCount, progress, estimatedTotalSize, canProcess, setTool, addFiles, addNativeFiles, addNativePaths, browseFiles, browseFolder, chooseOutput, loadDefaultOutputDirectory, openOutputDirectory, removeFile, clearFiles, process, cancel, retry, setOutputDir, estimateForFile }
})
