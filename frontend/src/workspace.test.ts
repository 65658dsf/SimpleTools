import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useWorkspaceStore } from './stores/workspace'

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

