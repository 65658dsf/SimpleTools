import { describe, expect, it } from 'vitest'
import QRCode from 'qrcode'
import { DEFAULT_QR_CODE, decodeBrowserQRCode, decodeQRCodePixels, normalizeQRCodeSettings, qrCodeByteLength, qrCodeOptions, qrCodeTabForKey, renderBrowserQRCodePreview } from './qrcode'

function renderQRCodePixels(text: string) {
  const code = QRCode.create(text, { errorCorrectionLevel: 'M' })
  const quietZone = 4
  const scale = 7
  const width = (code.modules.size + quietZone * 2) * scale
  const pixels = new Uint8ClampedArray(width * width * 4).fill(255)
  for (let row = 0; row < code.modules.size; row += 1) {
    for (let column = 0; column < code.modules.size; column += 1) {
      if (!code.modules.data[row * code.modules.size + column]) continue
      const startX = (column + quietZone) * scale
      const startY = (row + quietZone) * scale
      for (let y = startY; y < startY + scale; y += 1) {
        for (let x = startX; x < startX + scale; x += 1) {
          const offset = (y * width + x) * 4
          pixels[offset] = 0
          pixels[offset + 1] = 0
          pixels[offset + 2] = 0
        }
      }
    }
  }
  return { pixels, width }
}

describe('QR code options', () => {
  it('normalizes persisted settings to the native contract', () => {
    const settings = normalizeQRCodeSettings({
      text: 'https://example.com',
      size: 999,
      errorCorrection: 'unknown',
      foreground: 'red',
      background: '#ABCDEF',
      fileName: 'example',
    })

    expect(settings.size).toBe(DEFAULT_QR_CODE.size)
    expect(settings.errorCorrection).toBe(DEFAULT_QR_CODE.errorCorrection)
    expect(settings.foreground).toBe(DEFAULT_QR_CODE.foreground)
    expect(settings.background).toBe('#abcdef')
    expect(qrCodeOptions(settings)).not.toHaveProperty('fileName')
  })

  it('measures UTF-8 content bytes', () => {
    expect(qrCodeByteLength('SimpleTools')).toBe(11)
    expect(qrCodeByteLength('二维码')).toBe(9)
  })

  it('renders a bounded browser PNG preview', async () => {
    const preview = await renderBrowserQRCodePreview(qrCodeOptions({ ...DEFAULT_QR_CODE, text: 'preview' }), 320)
    expect(preview.size).toBe(320)
    expect(preview.dataUrl).toMatch(/^data:image\/png;base64,/)
  })

  it('rejects an empty browser preview', async () => {
    await expect(renderBrowserQRCodePreview({ ...qrCodeOptions(DEFAULT_QR_CODE), text: '  ' })).rejects.toThrow('required')
  })

  it('reports browser decode availability instead of pretending to decode', async () => {
    await expect(decodeBrowserQRCode('blob:qr-fixture')).rejects.toThrow(/unavailable/i)
  })

  it('decodes QR pixels in the browser fallback', () => {
    const text = 'https://simpletools.local/qr-decode'
    const { pixels, width } = renderQRCodePixels(text)
    expect(decodeQRCodePixels(pixels, width, width)).toBe(text)
  })

  it('preserves whitespace-only QR payloads', () => {
    const text = ' \n '
    const { pixels, width } = renderQRCodePixels(text)
    expect(decodeQRCodePixels(pixels, width, width)).toBe(text)
  })

  it('maps secondary tab keyboard navigation', () => {
    expect(qrCodeTabForKey('generate', 'ArrowRight')).toBe('decode')
    expect(qrCodeTabForKey('decode', 'ArrowLeft')).toBe('generate')
    expect(qrCodeTabForKey('decode', 'Home')).toBe('generate')
    expect(qrCodeTabForKey('generate', 'End')).toBe('decode')
    expect(qrCodeTabForKey('generate', 'Enter')).toBeUndefined()
  })
})
