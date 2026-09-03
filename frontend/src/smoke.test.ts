import { describe, expect, it } from 'vitest'
import { i18n } from './i18n'

const qrCodeLabels = [
  'qrcode',
  'qrcodeDesc',
  'bytes',
  'qrPreviewAlt',
  'qrGenerating',
  'qrPreviewFailed',
  'qrEmptyPreview',
  'qrContentLength',
  'qrByteLength',
  'qrCorrection',
  'qrSettings',
  'qrText',
  'qrTextPlaceholder',
  'qrSize',
  'qrCorrectionLow',
  'qrCorrectionMedium',
  'qrCorrectionQuartile',
  'qrCorrectionHigh',
  'qrColors',
  'resetColors',
  'foreground',
  'background',
  'fileName',
  'qrSaveFailed',
  'qrSaving',
  'qrSave',
  'qrGenerateTab',
  'qrDecodeTab',
  'qrDecodeDesc',
  'qrDecodeDropTitle',
  'qrDecodeDropHint',
  'qrDecodeBrowse',
  'qrDecodePreview',
  'qrDecodeEmpty',
  'qrDecoding',
  'qrDecodedText',
  'qrCopyText',
  'qrCopied',
  'qrDecodeFailed',
  'qrDecodeNoCode',
  'qrDecodeUnavailable',
  'qrDecodeImageOnly',
] as const

describe('frontend contract smoke tests', () => {
  it('contains every tool label in both supported locales', () => {
    const messages = i18n.global.messages.value
    for (const locale of ['en', 'zh'] as const) {
      expect(messages[locale].convert).toBeTruthy()
      expect(messages[locale].compress).toBeTruthy()
      expect(messages[locale].watermark).toBeTruthy()
      expect((messages[locale] as Record<string, string>).qrcode).toBeTruthy()
      expect(messages[locale].pdf).toBeTruthy()
    }
  })

  it('contains the home navigation label in both supported locales', () => {
    const messages = i18n.global.messages.value
    for (const locale of ['en', 'zh'] as const) {
      expect((messages[locale] as Record<string, string>).backHome).toBeTruthy()
    }
  })

  it('contains every QR code label in both supported locales', () => {
    const messages = i18n.global.messages.value
    for (const locale of ['en', 'zh'] as const) {
      const localized = messages[locale] as Record<string, string>
      for (const key of qrCodeLabels) expect(localized[key]).toBeTruthy()
    }
  })
})
