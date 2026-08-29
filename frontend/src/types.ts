export type ToolId = 'convert' | 'compress' | 'watermark' | 'qrcode' | 'pdf'
export type QueueStatus = 'queued' | 'processing' | 'done' | 'error' | 'cancelled'
export type ThemeMode = 'light' | 'dark' | 'system'

export type WatermarkPosition =
  | 'top-left'
  | 'top-center'
  | 'top-right'
  | 'center-left'
  | 'center'
  | 'center-right'
  | 'bottom-left'
  | 'bottom-center'
  | 'bottom-right'

export interface WatermarkOptions {
  text: string
  fontFamily: string
  fontSize: number
  color: string
  opacity: number
  position: WatermarkPosition
  margin: number
  rotation: number
  tile: boolean
  spacing: number
  shadow: boolean
}

export type QRCodeErrorCorrection = 'low' | 'medium' | 'quartile' | 'high'

export interface QRCodeOptions {
  text: string
  size: number
  errorCorrection: QRCodeErrorCorrection
  foreground: string
  background: string
}

export interface QRCodePreview {
  dataUrl: string
  size: number
}

export interface QRCodeDecodeResult {
  path: string
  width: number
  height: number
  format: string
  size: number
  text: string
  textBytes: number
  truncated?: boolean
}

export interface NativeInputFile {
  path: string
  name: string
  size: number
  kind: 'image' | 'pdf'
  relativePath?: string
}

export interface QueueFile {
  id: string
  file?: File
  path?: string
  name: string
  size: number
  type: string
	relativePath?: string
  status: QueueStatus
  progress: number
  error?: string
  warning?: string
  previewUrl?: string
  resultPreviewUrl?: string
  resultName?: string
  resultNames?: string[]
}

export interface JobRequest {
  tool: ToolId
  inputs: string[]
	inputRelativeDirs?: Record<string, string>
  outputDirectory: string
  format?: string
  quality?: number
  targetBytes?: number
  lossless?: boolean
  preserveMetadata?: boolean
  recursive?: boolean
  dpi?: number
  pageRange?: string
  watermark?: WatermarkOptions
}

export interface JobItem {
  id: string
  path: string
  name: string
  state: string
  progress: number
  output?: string
  outputs?: string[]
  error?: string
  warning?: string
}

export interface JobStatus {
  id: string
  state: string
  total: number
  completed: number
  failed: number
  progress: number
  current?: string
  error?: string
  outputs?: string[]
  items: JobItem[]
}

export interface PreviewOptions {
  maxDimension?: number
  maxPixels?: number
}

export interface Preview {
  path: string
  width: number
  height: number
  format: string
  size: number
  dataUrl?: string
  truncated?: boolean
}

export interface WatermarkPreview {
  path: string
  width: number
  height: number
  beforeDataUrl: string
  afterDataUrl: string
  truncated?: boolean
}

export interface WailsService {
  openInputFiles(): Promise<NativeInputFile[]>
  openInputFilesFromPaths(paths: string[]): Promise<NativeInputFile[]>
  openInputFolder(): Promise<NativeInputFile[]>
  chooseOutputDirectory(): Promise<string>
  getDefaultOutputDirectory(): Promise<string>
  openOutputDirectory(path: string): Promise<void>
  previewImage(path: string, options?: PreviewOptions): Promise<Preview>
  previewWatermark(path: string, watermark: WatermarkOptions, maxDimension?: number): Promise<WatermarkPreview>
  previewQRCode(options: QRCodeOptions, maxDimension?: number): Promise<QRCodePreview>
  saveQRCode(options: QRCodeOptions, outputDirectory: string, fileName: string): Promise<string>
  decodeQRCode(path: string): Promise<QRCodeDecodeResult>
  startJob(request: JobRequest): Promise<string>
  getJob(id: string): Promise<JobStatus>
  cancelJob(id: string): Promise<void>
	checkForUpdate(): Promise<UpdateInfo>
	downloadAndInstallUpdate(assetId: string): Promise<void>
	onUpdateAvailable(listener: (info: UpdateInfo) => void): () => void
	onUpdateProgress(listener: (progress: UpdateProgress) => void): () => void
  onJobProgress(listener: (status: JobStatus) => void): () => void
  onJobItem(listener: (item: JobItem) => void): () => void
  isNative(): boolean
}

export interface UpdateAsset {
  id: string
  version: string
  platform: string
  architecture: string
  url: string
  sha256: string
  signature: string
  size?: number
}

export interface UpdateInfo {
  available: boolean
  version: string
  url?: string
  notes?: string
  assetId?: string
  sha256?: string
  signature?: string
  assets?: UpdateAsset[]
}

export interface UpdateProgress {
  assetId: string
  state: string
  progress: number
}
