import type { JobItem, JobRequest, JobStatus, NativeInputFile, Preview, PreviewOptions, UpdateInfo, UpdateProgress, WailsService, WatermarkPreview } from '../types'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

type NativeMethod = (...args: unknown[]) => unknown

declare global {
  interface Window {
    go?: Record<string, Record<string, Record<string, NativeMethod>>>
    runtime?: {
      EventsOn?: (name: string, callback: (payload: unknown) => void) => (() => void) | void
      EventsOff?: (name: string) => void
    }
  }
}

function nativeApp(): Record<string, NativeMethod> | undefined {
  const root = window.go
  return root?.main?.App ?? root?.app?.App
}

function listen<T>(event: string, callback: (payload: T) => void): () => void {
  const eventsOn = window.runtime?.EventsOn
  if (!eventsOn) return () => undefined
  const off = eventsOn(event, payload => callback(payload as T))
  return typeof off === 'function' ? off : () => window.runtime?.EventsOff?.(event)
}

const nativeService: WailsService = {
  isNative: () => Boolean(nativeApp()),
  openInputFiles: () => App.OpenInputFiles() as Promise<NativeInputFile[]>,
  openInputFilesFromPaths: paths => App.OpenInputFilesFromPaths(paths) as Promise<NativeInputFile[]>,
  openInputFolder: () => App.OpenInputFolder() as Promise<NativeInputFile[]>,
  chooseOutputDirectory: () => App.ChooseOutputDirectory(),
  getDefaultOutputDirectory: () => App.GetDefaultOutputDirectory(),
  openOutputDirectory: path => App.OpenOutputDirectory(path),
  previewImage: (path, options = {}) => App.PreviewImage(path, options as never) as Promise<Preview>,
  previewWatermark: (path, watermark, maxDimension = 960) => App.PreviewWatermark(path, watermark, maxDimension) as Promise<WatermarkPreview>,
  startJob: request => App.StartJob(request as never),
  getJob: id => App.GetJob(id) as Promise<JobStatus>,
  cancelJob: id => App.CancelJob(id),
  checkForUpdate: () => App.CheckForUpdate() as Promise<UpdateInfo>,
  downloadAndInstallUpdate: assetId => App.DownloadAndInstallUpdate(assetId),
  onJobProgress: listener => nativeApp() ? EventsOn('job:progress', payload => listener(payload as JobStatus)) : listen<JobStatus>('job:progress', listener),
  onJobItem: listener => nativeApp() ? EventsOn('job:item', payload => listener(payload as JobItem)) : listen<JobItem>('job:item', listener),
  onUpdateAvailable: listener => nativeApp() ? EventsOn('update:available', payload => listener(payload as UpdateInfo)) : listen<UpdateInfo>('update:available', listener),
  onUpdateProgress: listener => nativeApp() ? EventsOn('update:progress', payload => listener(payload as UpdateProgress)) : listen<UpdateProgress>('update:progress', listener),
}

// Browser mode keeps the UI previewable during frontend development. The mock
// never pretends to write files; native mode always uses the Go job contract.
const browserService: WailsService = {
  isNative: () => false,
  async openInputFiles() { return [] },
  async openInputFilesFromPaths() { return [] },
  async openInputFolder() { return [] },
  async chooseOutputDirectory() { return '' },
  async getDefaultOutputDirectory() { return '' },
  async openOutputDirectory() { return undefined },
  async previewImage(path, options: PreviewOptions = {}) {
    const img = new Image()
    img.src = path
    await new Promise<void>((resolve, reject) => { img.onload = () => resolve(); img.onerror = () => reject(new Error('Unable to preview image')) })
    return { path, width: img.naturalWidth, height: img.naturalHeight, format: path.split('.').pop() ?? '', size: 0, dataUrl: path, ...options }
  },
  async previewWatermark() { throw new Error('Native watermark preview is unavailable') },
  async startJob(request) {
    await wait(500)
    return `browser-${Date.now()}-${request.tool}`
  },
  async getJob(id) {
    return { id, state: 'completed', total: 0, completed: 0, failed: 0, progress: 1, items: [] }
  },
  async cancelJob() { await wait(0) },
  async checkForUpdate() { return { available: false, version: '' } },
  async downloadAndInstallUpdate() { await wait(0) },
  onJobProgress() { return () => undefined },
  onJobItem() { return () => undefined },
  onUpdateAvailable() { return () => undefined },
  onUpdateProgress() { return () => undefined },
}

// Wails injects `window.go` only inside the packaged desktop runtime. Vite's
// browser preview also has a `window`, so checking the global alone would
// select the native adapter and make direct service calls fail with an
// unavailable bridge error.
export const wailsService: WailsService = typeof window === 'undefined' || !nativeApp() ? browserService : nativeService

function wait(ms: number) { return new Promise<void>(resolve => setTimeout(resolve, ms)) }
