import { describe, expect, it } from 'vitest'
import { i18n } from './i18n'

describe('frontend contract smoke tests', () => {
  it('contains the three tool labels in both supported locales', () => {
    const messages = i18n.global.messages.value
    for (const locale of ['en', 'zh'] as const) {
      expect(messages[locale].convert).toBeTruthy()
      expect(messages[locale].compress).toBeTruthy()
      expect(messages[locale].pdf).toBeTruthy()
    }
  })
})

