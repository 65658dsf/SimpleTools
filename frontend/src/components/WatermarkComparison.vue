<script setup lang="ts">
import { computed } from 'vue'
import { GripVertical, ImageOff, Loader2 } from 'lucide-vue-next'
import { comparisonSplitAfterKey } from '../watermark'

const props = defineProps<{
  beforeUrl?: string
  afterUrl?: string
  width?: number
  height?: number
  alt?: string
  beforeLabel: string
  afterLabel: string
  emptyLabel: string
  sliderLabel: string
  loadingLabel: string
  error?: string
  loading?: boolean
}>()

const split = defineModel<number>({ default: 50 })
const aspectRatio = computed(() => props.width && props.height ? `${props.width} / ${props.height}` : '16 / 10')
const maxWidth = computed(() => props.width && props.height ? `${Math.round(560 * props.width / props.height)}px` : '100%')

function onKeydown(event: KeyboardEvent) {
  const next = comparisonSplitAfterKey(split.value, event.key)
  if (next === undefined) return
  event.preventDefault()
  split.value = next
}
</script>

<template>
  <div class="comparison-stage" :class="{ empty: !beforeUrl }" :style="{ aspectRatio, maxWidth }">
    <template v-if="beforeUrl">
      <img class="comparison-image comparison-before" :src="beforeUrl" :alt="`${alt || ''} - ${beforeLabel}`" />
      <div v-if="afterUrl" class="comparison-after-clip" :style="{ clipPath: `inset(0 0 0 ${split}%)` }">
        <img class="comparison-image" :src="afterUrl" :alt="`${alt || ''} - ${afterLabel}`" />
      </div>
      <span class="comparison-label before-label">{{ beforeLabel }}</span>
      <span class="comparison-label after-label">{{ afterLabel }}</span>
      <div class="comparison-divider" :style="{ left: `${split}%` }" aria-hidden="true">
        <span class="comparison-handle"><GripVertical :size="17" /></span>
      </div>
      <input
        v-model.number="split"
        class="comparison-range"
        type="range"
        min="2"
        max="98"
        step="1"
        :aria-label="sliderLabel"
        @keydown="onKeydown"
      />
      <div v-if="loading" class="comparison-loading" role="status" :aria-label="loadingLabel"><Loader2 class="spin" :size="16" aria-hidden="true" /></div>
    </template>
    <div v-else class="comparison-empty" :role="loading ? 'status' : undefined">
      <Loader2 v-if="loading" class="spin" :size="24" aria-hidden="true" />
      <ImageOff v-else :size="24" />
      <span>{{ loading ? loadingLabel : emptyLabel }}</span>
    </div>
    <div v-if="error" class="comparison-error" role="status">{{ error }}</div>
  </div>
</template>
