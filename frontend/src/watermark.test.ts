import { afterEach, describe, expect, it, vi } from 'vitest'
import { createLatestTaskScheduler } from './latest-task'
import { applyWatermarkPreset, comparisonSplitAfterKey, DEFAULT_WATERMARK, matchingWatermarkPreset, normalizeWatermarkOptions, WATERMARK_PRESETS } from './watermark'

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>(done => { resolve = done })
  return { promise, resolve }
}

async function flushMicrotasks() {
  for (let index = 0; index < 5; index += 1) await Promise.resolve()
}

afterEach(() => vi.useRealTimers())

describe('watermark options', () => {
  it('normalizes persisted values to the backend contract', () => {
    const normalized = normalizeWatermarkOptions({
      text: 'Copyright',
      fontFamily: 'noto-sans-sc',
      fontSize: 999,
      color: 'not-a-color',
      opacity: -2,
      position: 'outside',
      rotation: 999,
    })

    expect(normalized.text).toBe('Copyright')
    expect(normalized.fontSize).toBe(512)
    expect(normalized.color).toBe(DEFAULT_WATERMARK.color)
    expect(normalized.opacity).toBe(0.01)
    expect(normalized.position).toBe(DEFAULT_WATERMARK.position)
    expect(normalized.rotation).toBe(180)
  })

  it('replaces unsupported persisted fonts with the bundled font', () => {
    expect(normalizeWatermarkOptions({ fontFamily: 'Comic Sans' }).fontFamily).toBe('noto-sans-sc')
    expect(normalizeWatermarkOptions({ fontFamily: 'sans-serif' }).fontFamily).toBe('noto-sans-sc')
  })

  it('applies presets without replacing custom text or font', () => {
    const source = { ...DEFAULT_WATERMARK, text: 'My studio', fontFamily: 'noto-sans-sc' }
    const result = applyWatermarkPreset(source, WATERMARK_PRESETS[2])

    expect(result.text).toBe('My studio')
    expect(result.fontFamily).toBe('noto-sans-sc')
    expect(result.tile).toBe(true)
    expect(matchingWatermarkPreset(result)).toBe('diagonal')
  })

  it('moves and clamps the comparison split from keyboard commands', () => {
    expect(comparisonSplitAfterKey(50, 'ArrowLeft')).toBe(49)
    expect(comparisonSplitAfterKey(50, 'PageUp')).toBe(60)
    expect(comparisonSplitAfterKey(50, 'Home')).toBe(2)
    expect(comparisonSplitAfterKey(97, 'ArrowRight')).toBe(98)
    expect(comparisonSplitAfterKey(50, 'Escape')).toBeUndefined()
  })

  it('debounces queued preview inputs and starts only the latest one', async () => {
    vi.useFakeTimers()
    const started: string[] = []
    const applied: string[] = []
    const scheduler = createLatestTaskScheduler<string, string>({
      debounceMs: 30,
      async task(input) {
        started.push(input)
        return input
      },
      onSuccess: result => applied.push(result),
      onError: () => undefined,
    })

    scheduler.schedule('first')
    await vi.advanceTimersByTimeAsync(20)
    scheduler.schedule('latest')
    await vi.advanceTimersByTimeAsync(29)
    expect(started).toEqual([])

    await vi.advanceTimersByTimeAsync(1)
    expect(started).toEqual(['latest'])
    expect(applied).toEqual(['latest'])
    scheduler.dispose()
  })

  it('suppresses stale results and never runs preview tasks concurrently', async () => {
    vi.useFakeTimers()
    const pending = new Map<string, ReturnType<typeof deferred<string>>>()
    const started: string[] = []
    const applied: string[] = []
    let active = 0
    let maxActive = 0
    const scheduler = createLatestTaskScheduler<string, string>({
      debounceMs: 10,
      task(input) {
        started.push(input)
        active += 1
        maxActive = Math.max(maxActive, active)
        const task = deferred<string>()
        pending.set(input, task)
        return task.promise.finally(() => { active -= 1 })
      },
      onSuccess: result => applied.push(result),
      onError: () => undefined,
    })

    scheduler.schedule('old')
    await vi.advanceTimersByTimeAsync(10)
    expect(started).toEqual(['old'])
    expect(active).toBe(1)

    scheduler.schedule('new')
    await vi.advanceTimersByTimeAsync(10)
    expect(started).toEqual(['old'])
    expect(active).toBe(1)

    pending.get('old')?.resolve('stale result')
    await flushMicrotasks()
    expect(started).toEqual(['old', 'new'])
    expect(applied).toEqual([])
    expect(maxActive).toBe(1)

    pending.get('new')?.resolve('latest result')
    await flushMicrotasks()
    expect(applied).toEqual(['latest result'])
    expect(active).toBe(0)
    expect(maxActive).toBe(1)
    scheduler.dispose()
  })
})
