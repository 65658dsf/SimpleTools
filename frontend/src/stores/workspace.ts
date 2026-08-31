import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import type { JobItem, JobProgress, JobRequest, JobStatus, NativeInputFile, QueueFile, RecentJobSummary, ToolId, WatermarkOptions } from '../types'
import { wailsService } from '../services/wails'
import { DEFAULT_QR_CODE, normalizeQRCodeSettings, type QRCodeSettings } from '../qrcode'
import { DEFAULT_WATERMARK, normalizeWatermarkOptions } from '../watermark'

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
export type QueueFilter = 'all' | 'queued' | 'processing' | 'done' | 'error'

const BYTES_PER_TARGET_UNIT: Record<TargetBytesUnit, number> = {
  B: 1,
  KB: 1024,
  MB: 1024 * 1024,
  GB: 1024 * 1024 * 1024,
}

const MAX_TARGET_BYTES = Number.MAX_SAFE_INTEGER
const PREVIEW_CONCURRENCY = 4
let activePreviewTasks = 0
const queuedPreviewTasks: Array<() => void> = []

// Native thumbnail decoding allocates image buffers in Go. Keep a small,
// process-wide queue so dropping a large folder does not create one IPC call
// and one pixel allocation per file at the same time.
async function withPreviewSlot<Result>(task: () => Promise<Result>): Promise<Result> {
  if (activePreviewTasks >= PREVIEW_CONCURRENCY) {
    await new Promise<void>(resolve => queuedPreviewTasks.push(resolve))
  }
  activePreviewTasks += 1
  try {
    return await task()
  } finally {
    activePreviewTasks -= 1
    queuedPreviewTasks.shift()?.()
  }
}

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
  watermark: WatermarkOptions
  qrCode: QRCodeSettings
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
  watermark: { ...DEFAULT_WATERMARK },
  qrCode: { ...DEFAULT_QR_CODE },
}

function defaultSettings(): WorkspaceSettings {
  return { ...DEFAULT_SETTINGS, watermark: { ...DEFAULT_WATERMARK }, qrCode: { ...DEFAULT_QR_CODE } }
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

const RECENT_JOBS_STORAGE_KEY = 'simpletools-recent-jobs'
const MAX_RECENT_JOBS = 20
const TOOL_IDS: ToolId[] = ['convert', 'compress', 'watermark', 'qrcode', 'pdf']

function isToolId(value: unknown): value is ToolId {
  return typeof value === 'string' && TOOL_IDS.includes(value as ToolId)
}

function cloneJobRequest(request: JobRequest): JobRequest {
  return {
    ...request,
    inputs: [...request.inputs],
    inputRelativeDirs: request.inputRelativeDirs ? { ...request.inputRelativeDirs } : undefined,
    watermark: request.watermark ? { ...request.watermark } : undefined,
  }
}

function isPersistedJobRequest(value: unknown): value is JobRequest {
  if (!value || typeof value !== 'object') return false
  const request = value as Partial<JobRequest>
  return isToolId(request.tool)
    && Array.isArray(request.inputs)
    && request.inputs.every(input => typeof input === 'string')
    && typeof request.outputDirectory === 'string'
}

function readRecentJobs(): RecentJobSummary[] {
  const raw = readStorage(RECENT_JOBS_STORAGE_KEY)
  if (!raw) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.map(value => {
      if (!value || typeof value !== 'object') return undefined
      const entry = value as Partial<RecentJobSummary>
      if (entry.cancelled === undefined) return { ...entry, cancelled: 0 }
      return value
    }).filter((value): value is RecentJobSummary => {
      if (!value || typeof value !== 'object') return false
      const entry = value as Partial<RecentJobSummary>
      return typeof entry.id === 'string'
        && isToolId(entry.tool)
        && typeof entry.total === 'number' && Number.isFinite(entry.total) && entry.total >= 0
        && typeof entry.completed === 'number' && Number.isFinite(entry.completed) && entry.completed >= 0
        && typeof entry.failed === 'number' && Number.isFinite(entry.failed) && entry.failed >= 0
        && typeof entry.cancelled === 'number' && Number.isFinite(entry.cancelled) && entry.cancelled >= 0
        && typeof entry.outputDirectory === 'string'
        && typeof entry.finishedAt === 'string'
        && Array.isArray(entry.inputPaths)
        && entry.inputPaths.every(input => typeof input === 'string')
        && (entry.request === undefined || isPersistedJobRequest(entry.request))
    }).slice(0, MAX_RECENT_JOBS)
  } catch {
    return []
  }
}

function writeRecentJobs(jobs: RecentJobSummary[]) {
  writeStorage(RECENT_JOBS_STORAGE_KEY, JSON.stringify(jobs))
}

function readSettings(): WorkspaceSettings {
  const raw = readStorage('simpletools-settings')
  if (!raw) return defaultSettings()
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
      watermark: normalizeWatermarkOptions(saved.watermark),
      qrCode: normalizeQRCodeSettings(saved.qrCode),
    }
  } catch {
    return defaultSettings()
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
  const queueFilter = ref<QueueFilter>('all')
  const recentJobs = ref<RecentJobSummary[]>(readRecentJobs())
  let stopProgress: () => void = () => undefined
  let stopItems: () => void = () => undefined

  const completeCount = computed(() => files.value.filter(item => item.status === 'done').length)
  const visibleFiles = computed(() => queueFilter.value === 'all'
    ? files.value
    : files.value.filter(item => item.status === queueFilter.value))
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
  // `recursive` controls folder expansion when files are added. Once a file
  // is in the queue it is an explicit input, so changing the toggle must not
  // leave a visible row that can never be processed.
  const isEligible = (_item: QueueFile) => true
  const canProcess = computed(() => {
    const hasValidWatermark = activeTool.value !== 'watermark' || Boolean(settings.value.watermark.text.trim())
    return !running.value && hasValidWatermark && files.value.some(item => isEligible(item) && (item.status === 'queued' || item.status === 'error'))
  })

  function autoSelectTargetUnit() {
    if (!settings.value.targetBytesUnitAuto || activeTool.value !== 'compress') return
    const largestInput = files.value.reduce((largest, item) => Math.max(largest, item.size), 0)
    if (largestInput <= 0) return
    settings.value.targetBytesUnit = targetUnitForBytes(largestInput)
  }

  watch(outputDir, value => writeStorage('simpletools-output', value))
  watch(settings, value => writeStorage('simpletools-settings', JSON.stringify(value)), { deep: true })
  watch(recentJobs, value => writeRecentJobs(value), { deep: true })

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

  function setTool(tool: ToolId): boolean {
    if (tool !== 'convert' && tool !== 'compress' && tool !== 'watermark' && tool !== 'qrcode' && tool !== 'pdf') return false
    // Keep the active queue attached to its running job. Navigation controls
    // call this method before changing routes, so a false result also lets
    // callers keep the user on the current workspace while processing.
    if (running.value && tool !== activeTool.value) return false
    activeTool.value = tool
    if (!running.value) {
      files.value = files.value.filter(item => activeTool.value === 'pdf' ? item.type === 'application/pdf' : item.type.startsWith('image/'))
    }
    autoSelectTargetUnit()
    return true
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
      if (item.file) url = await withPreviewSlot(() => browserThumbnail(item.file as File))
      else if (item.path) url = await withPreviewSlot(async () => (await wailsService.previewImage(item.path as string, { maxDimension: 240 })).dataUrl ?? '')
      // A queued preview may finish after the row was removed or reset for a
      // retry. Avoid retaining a detached result or replacing a newer preview.
      if (!files.value.some(entry => entry.id === item.id) || (target && item.status !== 'done')) return
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

  function recordRecentJob(processed: QueueFile[], request: JobRequest) {
    const terminal = processed.filter(item => item.status === 'done' || item.status === 'error' || item.status === 'cancelled')
    if (!terminal.length) return
    const summary: RecentJobSummary = {
      id: id(),
      tool: request.tool,
      total: processed.length,
      completed: terminal.filter(item => item.status === 'done').length,
      failed: terminal.filter(item => item.status === 'error').length,
      cancelled: terminal.filter(item => item.status === 'cancelled').length,
      outputDirectory: request.outputDirectory,
      finishedAt: new Date().toISOString(),
      inputPaths: [...request.inputs],
      request: cloneJobRequest(request),
    }
    recentJobs.value = [summary, ...recentJobs.value.filter(item => item.id !== summary.id)].slice(0, MAX_RECENT_JOBS)
    writeRecentJobs(recentJobs.value)
  }

  function removeRecentJob(jobId: string) {
    recentJobs.value = recentJobs.value.filter(item => item.id !== jobId)
    writeRecentJobs(recentJobs.value)
  }

  function clearRecentJobs() {
    recentJobs.value = []
    writeRecentJobs(recentJobs.value)
  }

  async function openRecentOutputDirectory(job: RecentJobSummary): Promise<{ ok: boolean, message?: string }> {
    if (!wailsService.isNative()) return { ok: false, message: 'desktop-only' }
    const directory = (job.outputDirectory || job.request?.outputDirectory || '').trim()
    if (!directory) return { ok: false, message: 'missing-output' }
    try {
      await wailsService.openOutputDirectory(directory)
      return { ok: true }
    } catch (error) {
      return { ok: false, message: error instanceof Error ? error.message : 'open-failed' }
    }
  }

  function restoreRequestSettings(request: JobRequest) {
    setTool(request.tool)
    if (request.outputDirectory !== undefined) outputDir.value = request.outputDirectory
    if (request.format !== undefined) settings.value.format = request.format
    if (request.quality !== undefined) settings.value.quality = request.quality
    if (request.tool === 'compress') {
      if (request.targetBytes !== undefined) {
        settings.value.targetBytes = request.targetBytes
        settings.value.targetBytesUnit = targetUnitForBytes(request.targetBytes)
        settings.value.targetBytesUnitAuto = false
      } else {
        settings.value.targetBytes = 0
        settings.value.targetBytesUnitAuto = true
      }
    }
    if (request.lossless !== undefined) settings.value.lossless = request.lossless
    if (request.preserveMetadata !== undefined) settings.value.preserveMetadata = request.preserveMetadata
    if (request.recursive !== undefined) settings.value.recursive = request.recursive
    if (request.dpi !== undefined) settings.value.dpi = request.dpi
    if (request.pageRange !== undefined) settings.value.pageRange = request.pageRange
    if (request.watermark) settings.value.watermark = normalizeWatermarkOptions(request.watermark)
  }

  async function rerunRecentJob(job: RecentJobSummary): Promise<{ ok: boolean, message?: string }> {
    if (!wailsService.isNative()) return { ok: false, message: 'desktop-only' }
    const inputPaths = job.request?.inputs?.length ? job.request.inputs : job.inputPaths
    if (!inputPaths?.length) return { ok: false, message: 'inputs-unavailable' }
    if (running.value) return { ok: false, message: 'processing' }

    // Older summaries predate the persisted request snapshot. They can still
    // be rerun with the tool's current defaults while preserving the recorded
    // input paths and output directory.
    const request = job.request
      ? cloneJobRequest({ ...job.request, inputs: [...inputPaths] })
      : { tool: job.tool, inputs: [...inputPaths], outputDirectory: job.outputDirectory }

    let selected: NativeInputFile[]
    try {
      selected = await wailsService.openInputFilesFromPaths(inputPaths)
    } catch (error) {
      return { ok: false, message: error instanceof Error ? error.message : 'inputs-unavailable' }
    }
    if (!selected.length) return { ok: false, message: 'inputs-unavailable' }
    const relativeDirs = request.inputRelativeDirs ?? {}
    selected = selected.map(file => ({ ...file, relativePath: relativeDirs[file.path] ?? file.relativePath }))
    clearFiles()
    restoreRequestSettings(request)
    addNativeFiles(selected)
    if (!files.value.length) return { ok: false, message: 'inputs-unavailable' }
    void process()
    return { ok: true }
  }

  function removeFile(fileId: string) { const item = files.value.find(entry => entry.id === fileId); if (!item || item.status === 'processing') return; releasePreview(item); files.value = files.value.filter(entry => entry.id !== fileId); autoSelectTargetUnit() }
  function clearFiles() { if (running.value) return; files.value.forEach(releasePreview); files.value = []; autoSelectTargetUnit() }
  function resetForRetry(item: QueueFile) {
    releaseResultPreview(item)
    item.status = 'queued'
    item.progress = 0
    item.error = undefined
    item.warning = undefined
    item.resultName = undefined
    item.resultNames = undefined
    item.resultPreviewUrl = undefined
    item.originalBytes = undefined
    item.compressedBytes = undefined
  }
  function retry(fileId: string) { const item = files.value.find(entry => entry.id === fileId); if (item) resetForRetry(item) }
  function retryFailed() { if (running.value) return; files.value.filter(item => item.status === 'error').forEach(resetForRetry) }
  function clearCompleted() { if (running.value) return; files.value.filter(item => item.status === 'done').forEach(releasePreview); files.value = files.value.filter(item => item.status !== 'done'); autoSelectTargetUnit() }
  function setOutputDir(dir: string) { outputDir.value = dir }
  function estimateForFile(item: QueueFile) {
    return estimateCompressedSize(item.size, settings.value.quality, settings.value.targetBytes, settings.value.lossless, item.type)
  }
  function compressionSavings(item: QueueFile): number | undefined {
    if (item.originalBytes === undefined || item.compressedBytes === undefined || item.originalBytes <= 0) return undefined
    return Math.round((1 - item.compressedBytes / item.originalBytes) * 100)
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
      watermark: activeTool.value === 'watermark' ? { ...settings.value.watermark } : undefined,
      recursive: settings.value.recursive,
      preserveMetadata: settings.value.preserveMetadata,
    }
  }

  async function process() {
    const processable = files.value.filter(item => isEligible(item) && (item.status === 'queued' || item.status === 'error'))
    if (!processable.length || running.value || (activeTool.value === 'watermark' && !settings.value.watermark.text.trim())) return
    running.value = true
    cancelRequested.value = false
    if (wailsService.isNative()) await loadDefaultOutputDirectory()
    const requestSnapshot = cloneJobRequest(nativeRequest(processable))
    processable.forEach(item => { item.status = 'processing'; item.progress = 0; item.error = undefined })
    if (wailsService.isNative() && processable.every(item => item.path)) {
      let latestState: string
      let lastEventAt: number
      let wakeEvent: (() => void) | undefined
      const waitForNativeEvent = (timeoutMs: number) => new Promise<boolean>(resolve => {
        let settled = false
        const timerRef: { id?: ReturnType<typeof setTimeout> } = {}
        const finish = (eventReceived: boolean) => {
          if (settled) return
          settled = true
          if (timerRef.id !== undefined) clearTimeout(timerRef.id)
          if (wakeEvent === notify) wakeEvent = undefined
          resolve(eventReceived)
        }
        const notify = () => finish(true)
        wakeEvent = notify
        timerRef.id = setTimeout(() => finish(false), Math.max(1, timeoutMs))
      })
      const onProgress = (status: JobProgress | JobStatus) => {
        if (!activeJobId.value || status.id !== activeJobId.value) return
        latestState = status.state
        lastEventAt = Date.now()
        applyStatus(status)
        wakeEvent?.()
      }
      const onItem = (item: JobItem) => {
        if (!activeJobId.value || !item.id.startsWith(`${activeJobId.value}-item-`)) return
        lastEventAt = Date.now()
        applyItem(item)
        wakeEvent?.()
      }
      try {
        stopProgress = wailsService.onJobProgress(onProgress)
        stopItems = wailsService.onJobItem(onItem)
        activeJobId.value = await wailsService.startJob(requestSnapshot)
        if (cancelRequested.value) {
          try { await wailsService.cancelJob(activeJobId.value) } catch { /* status polling still determines the final state */ }
        }
        let status = await wailsService.getJob(activeJobId.value)
        latestState = status.state
        lastEventAt = Date.now()
        applyStatus(status)
        while (latestState === 'queued' || latestState === 'running') {
          const eventReceived = await waitForNativeEvent(Math.max(1, 5000 - (Date.now() - lastEventAt)))
          if (latestState !== 'queued' && latestState !== 'running') break
          if (!eventReceived) {
            // Events are authoritative during normal processing. A single
            // five-second watchdog poll recovers from a dropped event without
            // producing a continuous IPC stream for every file update.
            status = await wailsService.getJob(activeJobId.value)
            latestState = status.state
            lastEventAt = Date.now()
            applyStatus(status)
          }
        }
        // A terminal lightweight event may arrive even when an earlier item
        // event was dropped by the bridge. Reconcile once with the complete
        // snapshot so no row remains stuck in the processing state.
        if (latestState !== 'queued' && latestState !== 'running') {
          try {
            const finalStatus = await wailsService.getJob(activeJobId.value)
            applyStatus(finalStatus)
          } catch {
            // The terminal event is still useful when the job was evicted or
            // the bridge closes during application shutdown.
          }
        }
      } catch (error) {
        files.value.forEach(item => { if (item.status === 'processing') { item.status = 'error'; item.error = error instanceof Error ? error.message : 'Processing failed' } })
      } finally {
        stopProgress(); stopItems(); stopProgress = () => undefined; stopItems = () => undefined; activeJobId.value = undefined; running.value = false
        cancelRequested.value = false
        recordRecentJob(processable, requestSnapshot)
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
      item.resultName = activeTool.value === 'pdf' ? `${stripExtension(item.name)}-page-001.png` : activeTool.value === 'compress' ? `${stripExtension(item.name)}-compressed${extension(item.name)}` : activeTool.value === 'watermark' ? `${stripExtension(item.name)}-watermarked${extension(item.name)}` : `${stripExtension(item.name)}.${settings.value.format}`
      if (activeTool.value === 'compress') {
        item.originalBytes = item.size
        item.compressedBytes = estimateForFile(item)
      }
      item.progress = 100; item.status = 'done'
    }
    if (cancelRequested.value) {
      processable
        .filter(item => item.status === 'processing' || item.status === 'queued')
        .forEach(item => { item.status = 'cancelled'; item.progress = 0 })
    }
    recordRecentJob(processable, requestSnapshot)
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

  function applyStatus(status: JobProgress | JobStatus) {
    if ('items' in status) status.items?.forEach(applyItem)
    if ('item' in status && status.item) applyItem(status.item)
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
    target.originalBytes = item.originalBytes
    target.compressedBytes = item.compressedBytes
    target.resultName = item.output?.split(/[\\/]/).pop()
    target.resultNames = item.outputs
    if (item.output && activeTool.value !== 'pdf') void loadPreview(target, true)
  }

  async function simulateProgress(item: QueueFile) { for (const value of [20, 45, 70, 90]) { await new Promise(resolve => setTimeout(resolve, 100)); if (cancelRequested.value) return false; item.progress = value } return true }

  return { activeTool, files, visibleFiles, queueFilter, recentJobs, outputDir, running, settings, targetValue, targetUnit, targetUnitOptions: TARGET_BYTES_UNITS, completeCount, progress, estimatedTotalSize, canProcess, setTool, addFiles, addNativeFiles, addNativePaths, browseFiles, browseFolder, chooseOutput, loadDefaultOutputDirectory, openOutputDirectory, openRecentOutputDirectory, removeRecentJob, clearRecentJobs, rerunRecentJob, removeFile, clearFiles, retry, retryFailed, clearCompleted, process, cancel, setOutputDir, estimateForFile, compressionSavings }
})
