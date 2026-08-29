import { describe, expect, it } from 'vitest'
import { DEFAULT_QR_CODE, normalizeQRCodeSettings, qrCodeByteLength, qrCodeOptions, renderBrowserQRCodePreview } from './qrcode'

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
})
