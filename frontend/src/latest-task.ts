export interface LatestTaskSchedulerOptions<Input, Result> {
  debounceMs: number
  task: (input: Input) => Promise<Result>
  onSuccess: (result: Result, input: Input) => void
  onError: (error: unknown, input: Input) => void
}

export interface LatestTaskScheduler<Input> {
  schedule(input: Input): void
  cancel(): void
  dispose(): void
}

/** Debounce inputs, run one task at a time, and publish only the latest result. */
export function createLatestTaskScheduler<Input, Result>(options: LatestTaskSchedulerOptions<Input, Result>): LatestTaskScheduler<Input> {
  const delay = Math.max(0, options.debounceMs)
  let generation = 0
  let timer: ReturnType<typeof setTimeout> | undefined
  let nextInput: Input
  let hasInput = false
  let ready = false
  let running = false
  let disposed = false

  function clearTimer() {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }

  function schedule(input: Input) {
    if (disposed) return
    generation += 1
    nextInput = input
    hasInput = true
    ready = false
    clearTimer()
    timer = setTimeout(() => {
      timer = undefined
      ready = true
      void pump()
    }, delay)
  }

  function cancel() {
    if (disposed) return
    generation += 1
    hasInput = false
    ready = false
    clearTimer()
  }

  async function pump() {
    if (disposed || running || !ready || !hasInput) return
    const input = nextInput
    const taskGeneration = generation
    hasInput = false
    ready = false
    running = true

    let result: Result | undefined
    let taskError: unknown
    let failed = false
    try {
      result = await options.task(input)
    } catch (error) {
      failed = true
      taskError = error
    }

    try {
      if (!disposed && taskGeneration === generation) {
        if (failed) options.onError(taskError, input)
        else options.onSuccess(result as Result, input)
      }
    } finally {
      running = false
      if (ready) void pump()
    }
  }

  function dispose() {
    if (disposed) return
    generation += 1
    disposed = true
    hasInput = false
    ready = false
    clearTimer()
  }

  return { schedule, cancel, dispose }
}
