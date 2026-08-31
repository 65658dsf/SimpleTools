import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { DEFAULT_QR_CODE } from './qrcode'
import { estimateCompressedSize, targetBytesToValue, targetUnitForBytes, targetValueToBytes, useWorkspaceStore } from './stores/workspace'

class MemoryStorage implements Storage {
  private readonly values = new Map<string, string>()

  get length() { return this.values.size }
  clear() { this.values.clear() }
  getItem(key: string) { return this.values.get(key) ?? null }
  key(index: number) { return [...this.values.keys()][index] ?? null }
  removeItem(key: string) { this.values.delete(key) }
  setItem(key: string, value: string) { this.values.set(key, value) }
}

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

  it('does not switch tools while a queue is processing', async () => {
    const store = useWorkspaceStore()
    store.addFiles([imageFile('one.png')])
    const processing = store.process()
    await new Promise(resolve => setTimeout(resolve, 20))

    expect(store.setTool('pdf')).toBe(false)
    expect(store.activeTool).toBe('convert')
    expect(store.files).toHaveLength(1)

    await store.cancel()
    await processing
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

describe('QR code settings', () => {
  let storage: MemoryStorage

  beforeEach(() => {
    storage = new MemoryStorage()
    vi.stubGlobal('localStorage', storage)
    setActivePinia(createPinia())
  })

  afterEach(() => vi.unstubAllGlobals())

  it('persists nested settings and restores them in a new store', async () => {
    const expected = {
      text: 'https://simpletools.example/qr',
      size: 1024,
      errorCorrection: 'high' as const,
      foreground: '#123456',
      background: '#fedcba',
      fileName: 'simpletools-link',
    }
    const first = useWorkspaceStore()
    Object.assign(first.settings.qrCode, expected)

    await nextTick()

    const saved = JSON.parse(storage.getItem('simpletools-settings') ?? '{}')
    expect(saved.qrCode).toEqual(expected)

    setActivePinia(createPinia())
    expect(useWorkspaceStore().settings.qrCode).toEqual(expected)
  })

  it('normalizes persisted QR code settings through the workspace store', () => {
    storage.setItem('simpletools-settings', JSON.stringify({
      qrCode: {
        text: 'saved content',
        size: 999,
        errorCorrection: 'maximum',
        foreground: 'black',
        background: '#AABBCC',
        fileName: 'x'.repeat(200),
      },
    }))

    expect(useWorkspaceStore().settings.qrCode).toEqual({
      ...DEFAULT_QR_CODE,
      text: 'saved content',
      background: '#aabbcc',
      fileName: 'x'.repeat(160),
    })
  })

  it('does not share nested QR code defaults between store instances', () => {
    vi.stubGlobal('localStorage', undefined)
    const first = useWorkspaceStore()
    first.settings.qrCode.text = 'changed'

    setActivePinia(createPinia())
    const second = useWorkspaceStore()

    expect(second.settings.qrCode).toEqual(DEFAULT_QR_CODE)
    expect(second.settings.qrCode).not.toBe(first.settings.qrCode)
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

describe('recent jobs', () => {
  let storage: MemoryStorage

  beforeEach(() => {
    storage = new MemoryStorage()
    vi.stubGlobal('localStorage', storage)
    setActivePinia(createPinia())
  })

  afterEach(() => vi.unstubAllGlobals())

  it('records a completed browser run and restores its summary', async () => {
    const store = useWorkspaceStore()
    store.setTool('compress')
    store.addFiles([imageFile('photo.png')])

    await store.process()
    await nextTick()

    expect(store.recentJobs).toHaveLength(1)
    expect(store.recentJobs[0]).toMatchObject({
      tool: 'compress',
      total: 1,
      completed: 1,
      failed: 0,
      cancelled: 0,
      outputDirectory: '',
      inputPaths: [],
    })
    expect(JSON.parse(storage.getItem('simpletools-recent-jobs') ?? '[]')).toHaveLength(1)

    setActivePinia(createPinia())
    expect(useWorkspaceStore().recentJobs[0].tool).toBe('compress')
  })

  it('explains when a browser-only recent job cannot be rerun', async () => {
    const store = useWorkspaceStore()
    store.addFiles([imageFile('photo.png')])
    await store.process()

    const result = await store.rerunRecentJob(store.recentJobs[0])
    expect(result).toEqual({ ok: false, message: 'desktop-only' })
  })

  it('keeps at most twenty persisted recent jobs', () => {
    storage.setItem('simpletools-recent-jobs', JSON.stringify(Array.from({ length: 25 }, (_, index) => ({
      id: `job-${index}`,
      tool: 'convert',
      total: 1,
      completed: 1,
      failed: 0,
      cancelled: 0,
      outputDirectory: '',
      finishedAt: new Date(index).toISOString(),
      inputPaths: [],
      request: { tool: 'convert', inputs: [], outputDirectory: '' },
    }))))

    const store = useWorkspaceStore()
    expect(store.recentJobs).toHaveLength(20)
    expect(store.recentJobs[0].id).toBe('job-0')
  })

  it('keeps legacy recent jobs that do not have a request snapshot', () => {
    storage.setItem('simpletools-recent-jobs', JSON.stringify([{
      id: 'legacy-job',
      tool: 'convert',
      total: 2,
      completed: 2,
      failed: 0,
      outputDirectory: '',
      finishedAt: new Date().toISOString(),
      inputPaths: ['photo-a.png', 'photo-b.png'],
    }]))

    const store = useWorkspaceStore()
    expect(store.recentJobs).toHaveLength(1)
    expect(store.recentJobs[0].request).toBeUndefined()
    expect(store.recentJobs[0].cancelled).toBe(0)
  })
})
