import QRCode from 'qrcode'
import type { QRCodeErrorCorrection, QRCodeOptions } from './types'

export const QR_CODE_SIZES = [256, 512, 1024, 2048] as const
export const QR_CODE_ERROR_CORRECTIONS: QRCodeErrorCorrection[] = ['low', 'medium', 'quartile', 'high']

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
