import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { estimateCompressedSize, useWorkspaceStore } from './stores/workspace'

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
