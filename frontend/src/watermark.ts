import type { WatermarkOptions, WatermarkPosition } from './types'

export type WatermarkPresetId = 'subtle' | 'centered' | 'diagonal' | 'dense'

export interface WatermarkPreset {
  id: WatermarkPresetId
  options: Omit<WatermarkOptions, 'text' | 'fontFamily'>
}

export const DEFAULT_WATERMARK: WatermarkOptions = {
  text: 'SimpleTools',
  fontFamily: 'noto-sans-sc',
  fontSize: 48,
  color: '#ffffff',
  opacity: 0.62,
  position: 'bottom-right',
  margin: 32,
  rotation: 0,
  tile: false,
  spacing: 180,
  shadow: true,
}

export const WATERMARK_PRESETS: WatermarkPreset[] = [
  {
    id: 'subtle',
    options: { fontSize: 36, color: '#ffffff', opacity: 0.55, position: 'bottom-right', margin: 28, rotation: 0, tile: false, spacing: 180, shadow: true },
  },
  {
    id: 'centered',
    options: { fontSize: 64, color: '#ffffff', opacity: 0.72, position: 'center', margin: 32, rotation: 0, tile: false, spacing: 180, shadow: true },
  },
  {
    id: 'diagonal',
    options: { fontSize: 52, color: '#ffffff', opacity: 0.3, position: 'center', margin: 24, rotation: -28, tile: true, spacing: 190, shadow: false },
  },
  {
    id: 'dense',
    options: { fontSize: 32, color: '#111827', opacity: 0.34, position: 'center', margin: 20, rotation: -24, tile: true, spacing: 100, shadow: false },
  },
]

const POSITIONS: WatermarkPosition[] = [
  'top-left', 'top-center', 'top-right',
  'center-left', 'center', 'center-right',
  'bottom-left', 'bottom-center', 'bottom-right',
]
const FONT_ALIASES = new Set(['noto-sans-sc', 'notosanssc', 'sans', 'sans-serif', 'system'])

function finiteNumber(value: unknown, fallback: number, min: number, max: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? Math.min(max, Math.max(min, value)) : fallback
}

export function normalizeWatermarkOptions(value: unknown): WatermarkOptions {
  if (!value || typeof value !== 'object') return { ...DEFAULT_WATERMARK }
  const saved = value as Partial<WatermarkOptions>
  const position = typeof saved.position === 'string' && POSITIONS.includes(saved.position as WatermarkPosition)
    ? saved.position as WatermarkPosition
    : DEFAULT_WATERMARK.position
  const color = typeof saved.color === 'string' && /^#[0-9a-f]{6}$/i.test(saved.color) ? saved.color : DEFAULT_WATERMARK.color
  const fontFamily = typeof saved.fontFamily === 'string' ? saved.fontFamily.trim().toLowerCase().replace(/[ _]/g, '-') : ''
  return {
    text: typeof saved.text === 'string' ? saved.text.slice(0, 2000) : DEFAULT_WATERMARK.text,
    fontFamily: FONT_ALIASES.has(fontFamily) ? 'noto-sans-sc' : DEFAULT_WATERMARK.fontFamily,
    fontSize: Math.round(finiteNumber(saved.fontSize, DEFAULT_WATERMARK.fontSize, 8, 512)),
    color,
    opacity: finiteNumber(saved.opacity, DEFAULT_WATERMARK.opacity, 0.01, 1),
    position,
    margin: Math.round(finiteNumber(saved.margin, DEFAULT_WATERMARK.margin, 0, 1000)),
    rotation: finiteNumber(saved.rotation, DEFAULT_WATERMARK.rotation, -180, 180),
    tile: typeof saved.tile === 'boolean' ? saved.tile : DEFAULT_WATERMARK.tile,
    spacing: Math.round(finiteNumber(saved.spacing, DEFAULT_WATERMARK.spacing, 0, 2000)),
    shadow: typeof saved.shadow === 'boolean' ? saved.shadow : DEFAULT_WATERMARK.shadow,
  }
}

export function applyWatermarkPreset(current: WatermarkOptions, preset: WatermarkPreset): WatermarkOptions {
  return { ...current, ...preset.options }
}

export function matchingWatermarkPreset(options: WatermarkOptions): WatermarkPresetId | undefined {
  return WATERMARK_PRESETS.find(preset => Object.entries(preset.options).every(([key, value]) => options[key as keyof WatermarkOptions] === value))?.id
}

export function comparisonSplitAfterKey(current: number, key: string): number | undefined {
  let next = current
  if (key === 'ArrowLeft' || key === 'ArrowDown') next -= 1
  else if (key === 'ArrowRight' || key === 'ArrowUp') next += 1
  else if (key === 'PageDown') next -= 10
  else if (key === 'PageUp') next += 10
  else if (key === 'Home') next = 2
  else if (key === 'End') next = 98
  else return undefined
  return Math.min(98, Math.max(2, next))
}

export interface BrowserWatermarkPreview {
  beforeDataUrl: string
  afterDataUrl: string
  width: number
  height: number
}

export async function renderBrowserWatermarkPreview(file: File, options: WatermarkOptions, maxDimension = 1400): Promise<BrowserWatermarkPreview> {
  const sourceUrl = URL.createObjectURL(file)
  try {
    const image = new Image()
    image.src = sourceUrl
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve()
      image.onerror = () => reject(new Error('Unable to decode image preview'))
    })

    const naturalWidth = Math.max(1, image.naturalWidth)
    const naturalHeight = Math.max(1, image.naturalHeight)
    const previewScale = Math.min(1, maxDimension / Math.max(naturalWidth, naturalHeight))
    const width = Math.max(1, Math.round(naturalWidth * previewScale))
    const height = Math.max(1, Math.round(naturalHeight * previewScale))
    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Canvas preview is unavailable')

    context.drawImage(image, 0, 0, width, height)
    const beforeDataUrl = canvas.toDataURL('image/png')
    drawWatermark(context, width, height, options, previewScale)
    return { beforeDataUrl, afterDataUrl: canvas.toDataURL('image/png'), width, height }
  } finally {
    URL.revokeObjectURL(sourceUrl)
  }
}

function drawWatermark(context: CanvasRenderingContext2D, width: number, height: number, options: WatermarkOptions, scale: number) {
  if (!options.text.trim()) return
  const fontSize = Math.max(4, options.fontSize * scale)
  const margin = options.margin * scale
  const spacing = options.spacing * scale

  context.save()
  context.globalAlpha = options.opacity
  context.fillStyle = options.color
  context.font = `${fontSize}px "${options.fontFamily}", "Segoe UI", sans-serif`
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  if (options.shadow) {
    context.shadowColor = 'rgba(0, 0, 0, .55)'
    context.shadowBlur = Math.max(1, 4 * scale)
    context.shadowOffsetX = Math.max(1, 2 * scale)
    context.shadowOffsetY = Math.max(1, 2 * scale)
  }

  const textWidth = Math.max(fontSize, context.measureText(options.text).width)
  if (options.tile) {
    const stepX = Math.max(1, textWidth + spacing)
    const stepY = Math.max(1, fontSize * 1.7 + spacing)
    for (let y = -height; y <= height * 2; y += stepY) {
      for (let x = -width; x <= width * 2; x += stepX) drawRotatedText(context, options.text, x, y, options.rotation)
    }
  } else {
    const [x, y] = positionCoordinates(options.position, width, height, textWidth, fontSize, margin)
    drawRotatedText(context, options.text, x, y, options.rotation)
  }
  context.restore()
}

function drawRotatedText(context: CanvasRenderingContext2D, text: string, x: number, y: number, rotation: number) {
  context.save()
  context.translate(x, y)
  context.rotate(rotation * Math.PI / 180)
  context.fillText(text, 0, 0)
  context.restore()
}

function positionCoordinates(position: WatermarkPosition, width: number, height: number, textWidth: number, fontSize: number, margin: number): [number, number] {
  const horizontal = position.endsWith('left') ? textWidth / 2 + margin : position.endsWith('right') ? width - textWidth / 2 - margin : width / 2
  const vertical = position.startsWith('top') ? fontSize / 2 + margin : position.startsWith('bottom') ? height - fontSize / 2 - margin : height / 2
  return [horizontal, vertical]
}
