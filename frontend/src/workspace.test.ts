import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { estimateCompressedSize, targetBytesToValue, targetUnitForBytes, targetValueToBytes, useWorkspaceStore } from './stores/workspace'

function imageFile(name: string) {
  return new File(['fixture'], name, { type: 'image/png' })
}

function pdfFile(name: string) {
  return new File(['fixture'], name, { type: 'application/pdf' })
}

describe('workspace queue behavior', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('filters queue entries when switching tools', () => {
    const store = useWorkspaceStore()
    store.addFiles([imageFile('photo.png'), pdfFile('document.pdf')])
    expect(store.files).toHaveLength(1)
    expect(store.files[0].name).toBe('photo.png')

    store.setTool('pdf')
    expect(store.files).toHaveLength(0)
    store.addFiles([pdfFile('document.pdf')])
    expect(store.files[0].type).toBe('application/pdf')
  })

  it('simulates browser processing and records a completed output', async () => {
    const store = useWorkspaceStore()
    store.addFiles([imageFile('photo.png')])
    expect(store.canProcess).toBe(true)

    await store.process()

    expect(store.files[0].status).toBe('done')
    expect(store.files[0].progress).toBe(100)
    expect(store.files[0].resultName).toBe('photo.webp')
  })

  it('keeps the source extension for a watermarked browser result', async () => {
    const store = useWorkspaceStore()
    store.setTool('watermark')
    store.addFiles([imageFile('photo.png')])

    await store.process()

    expect(store.files[0].status).toBe('done')
    expect(store.files[0].resultName).toBe('photo-watermarked.png')
  })

  it('requires non-empty watermark text before processing', async () => {
    const store = useWorkspaceStore()
    store.setTool('watermark')
    store.addFiles([imageFile('photo.png')])
    expect(store.canProcess).toBe(true)

    store.settings.watermark.text = '   '
    expect(store.canProcess).toBe(false)
    await store.process()
    expect(store.files[0].status).toBe('queued')
  })

  it('does not share nested watermark defaults between store instances', () => {
    const first = useWorkspaceStore()
    first.settings.watermark.text = 'changed'
    setActivePinia(createPinia())

    const second = useWorkspaceStore()
    expect(second.settings.watermark.text).toBe('SimpleTools')
  })

  it('marks in-flight browser work as cancelled', async () => {
    const store = useWorkspaceStore()
    store.addFiles([imageFile('one.png'), imageFile('two.png')])
    const processing = store.process()
    await new Promise(resolve => setTimeout(resolve, 60))
    await store.cancel()
    await processing

    expect(store.files.some(item => item.status === 'cancelled')).toBe(true)
    expect(store.running).toBe(false)
  })
})

describe('compression size estimate', () => {
  it('responds monotonically to quality and respects a target size', () => {
    const original = 100_000
    const lowQuality = estimateCompressedSize(original, 25)
    const highQuality = estimateCompressedSize(original, 90)

    expect(lowQuality).toBeLessThan(highQuality)
    expect(estimateCompressedSize(original, 76, 20_000)).toBe(20_000)
  })

  it('uses a stable metadata-removal estimate for lossless inputs', () => {
    expect(estimateCompressedSize(100_000, 10, 1_000, true)).toBe(92_000)
    expect(estimateCompressedSize(100_000, 10, 1_000, false, 'image/png')).toBe(92_000)
    expect(estimateCompressedSize(100_000, 10, 1_000, false, 'image/svg+xml')).toBe(92_000)
  })
})

describe('target size units', () => {
  it('selects binary units from the original input size', () => {
    expect(targetUnitForBytes(0)).toBe('B')
    expect(targetUnitForBytes(1023)).toBe('B')
    expect(targetUnitForBytes(1024)).toBe('KB')
    expect(targetUnitForBytes(1024 * 1024)).toBe('MB')
    expect(targetUnitForBytes(1024 * 1024 * 1024)).toBe('GB')
  })

  it('converts between the displayed value and backend bytes', () => {
    expect(targetValueToBytes(1.5, 'MB')).toBe(1.5 * 1024 * 1024)
    expect(targetBytesToValue(1.5 * 1024 * 1024, 'MB')).toBe(1.5)
    expect(targetValueToBytes(-1, 'KB')).toBe(0)
    expect(targetValueToBytes(Number.NaN, 'KB')).toBe(0)
  })

  it('auto-selects a unit until the user edits the target', () => {
    const store = useWorkspaceStore()
    store.setTool('compress')
    store.addNativeFiles([{ path: 'large.jpg', name: 'large.jpg', size: 3 * 1024 * 1024, kind: 'image' }])
    expect(store.targetUnit).toBe('MB')
    expect(store.targetValue).toBe(0)

    store.targetValue = 2
    expect(store.settings.targetBytes).toBe(2 * 1024 * 1024)
    store.addNativeFiles([{ path: 'tiny.jpg', name: 'tiny.jpg', size: 512, kind: 'image' }])
    expect(store.targetUnit).toBe('MB')

    store.targetUnit = 'KB'
    expect(store.targetUnit).toBe('KB')
    expect(store.targetValue).toBe(2048)
    expect(store.settings.targetBytes).toBe(2 * 1024 * 1024)
  })
})
