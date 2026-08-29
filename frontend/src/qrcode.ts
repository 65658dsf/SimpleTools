import QRCode from 'qrcode'
import jsQR from 'jsqr'
import type { QRCodeDecodeResult, QRCodeErrorCorrection, QRCodeOptions } from './types'

export const QR_CODE_SIZES = [256, 512, 1024, 2048] as const
export const QR_CODE_ERROR_CORRECTIONS: QRCodeErrorCorrection[] = ['low', 'medium', 'quartile', 'high']
export const QR_CODE_MAX_DECODE_PIXELS = 64 * 1024 * 1024
export type QRCodeTab = 'generate' | 'decode'

export interface QRCodeSettings extends QRCodeOptions {
  fileName: string
}

export const DEFAULT_QR_CODE: QRCodeSettings = {
  text: 'SimpleTools',
  size: 512,
  errorCorrection: 'medium',
  foreground: '#111827',
  background: '#ffffff',
  fileName: 'qrcode',
}

function validColor(value: unknown, fallback: string): string {
  return typeof value === 'string' && /^#[0-9a-f]{6}$/i.test(value) ? value.toLowerCase() : fallback
}

export function normalizeQRCodeSettings(value: unknown): QRCodeSettings {
  if (!value || typeof value !== 'object') return { ...DEFAULT_QR_CODE }
  const saved = value as Partial<QRCodeSettings>
  const size = typeof saved.size === 'number' && QR_CODE_SIZES.includes(saved.size as typeof QR_CODE_SIZES[number])
    ? saved.size
    : DEFAULT_QR_CODE.size
  const errorCorrection = typeof saved.errorCorrection === 'string' && QR_CODE_ERROR_CORRECTIONS.includes(saved.errorCorrection as QRCodeErrorCorrection)
    ? saved.errorCorrection as QRCodeErrorCorrection
    : DEFAULT_QR_CODE.errorCorrection
  return {
    text: typeof saved.text === 'string' ? saved.text.slice(0, 4096) : DEFAULT_QR_CODE.text,
    size,
    errorCorrection,
    foreground: validColor(saved.foreground, DEFAULT_QR_CODE.foreground),
    background: validColor(saved.background, DEFAULT_QR_CODE.background),
    fileName: typeof saved.fileName === 'string' ? saved.fileName.slice(0, 160) : DEFAULT_QR_CODE.fileName,
  }
}

export function qrCodeOptions(settings: QRCodeSettings): QRCodeOptions {
  const { fileName: _fileName, ...options } = settings
  return options
}

export function qrCodeByteLength(text: string): number {
  return new TextEncoder().encode(text).length
}

export function qrCodeTabForKey(active: QRCodeTab, key: string): QRCodeTab | undefined {
  if (key === 'ArrowLeft' || key === 'ArrowRight') return active === 'generate' ? 'decode' : 'generate'
  if (key === 'Home') return 'generate'
  if (key === 'End') return 'decode'
  return undefined
}

const ERROR_LEVELS: Record<QRCodeErrorCorrection, 'L' | 'M' | 'Q' | 'H'> = {
  low: 'L',
  medium: 'M',
  quartile: 'Q',
  high: 'H',
}

export async function renderBrowserQRCodePreview(options: QRCodeOptions, maxDimension = 512): Promise<{ dataUrl: string; size: number }> {
  if (!options.text.trim()) throw new Error('QR code text is required')
  const size = Math.max(64, Math.min(1024, Math.round(maxDimension), options.size))
  const dataUrl = await QRCode.toDataURL(options.text, {
    width: size,
    margin: 4,
    errorCorrectionLevel: ERROR_LEVELS[options.errorCorrection],
    color: { dark: options.foreground, light: options.background },
  })
  return { dataUrl, size }
}

export function decodeQRCodePixels(data: Uint8ClampedArray, width: number, height: number): string {
  if (width < 1 || height < 1 || data.length !== width * height * 4) throw new Error('Invalid QR code image pixels')
  const result = jsQR(data, width, height, { inversionAttempts: 'attemptBoth' })
  if (!result) throw new Error('No QR code found in image')
  if (!result.data) throw new Error('QR code contains no text')
  return result.data
}

/**
 * Decode a QR image in browser preview mode. Desktop mode uses the Go binding;
 * this fallback keeps browser development functional without sending selected
 * image data anywhere.
 */
export async function decodeBrowserQRCode(source: string): Promise<QRCodeDecodeResult> {
  if (!source) throw new Error('QR code image is required')
  if (typeof document === 'undefined' || typeof Image === 'undefined') {
    throw new Error('QR code decoding is unavailable in this environment')
  }

  const image = new Image()
  image.decoding = 'async'
  await new Promise<void>((resolve, reject) => {
    image.onload = () => resolve()
    image.onerror = () => reject(new Error('Unable to load QR code image'))
    image.src = source
  })

  const width = image.naturalWidth || image.width
  const height = image.naturalHeight || image.height
  if (!width || !height) throw new Error('QR code image has no pixels')
  const scale = Math.min(1, 4096 / Math.max(width, height))
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.round(width * scale))
  canvas.height = Math.max(1, Math.round(height * scale))
  const context = canvas.getContext('2d', { willReadFrequently: true })
  if (!context) throw new Error('Unable to inspect QR code image')
  context.drawImage(image, 0, 0, canvas.width, canvas.height)
  const pixels = context.getImageData(0, 0, canvas.width, canvas.height)
  const rawValue = decodeQRCodePixels(pixels.data, canvas.width, canvas.height)
  return {
    path: source,
    width,
    height,
    format: 'image',
    size: 0,
    text: rawValue,
    textBytes: new TextEncoder().encode(rawValue).length,
  }
}
