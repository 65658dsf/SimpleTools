<script setup lang="ts">
import { computed } from 'vue'

type ButtonVariant = 'default' | 'outline' | 'ghost'
type ButtonSize = 'default' | 'sm' | 'icon'

const props = withDefaults(defineProps<{
  variant?: ButtonVariant
  size?: ButtonSize
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), {
  variant: 'default',
  size: 'default',
  disabled: false,
  type: 'button',
})

// Keep the shadcn-vue variant/size contract local to this lightweight build.
// Existing theme classes can still be supplied by consumers and are merged by
// Vue onto the root button element.
const classes = computed(() => [
  'ui-button',
  `ui-button-${props.variant}`,
  `ui-button-${props.size}`,
])
</script>

<template>
  <button :type="props.type" :disabled="props.disabled" :class="classes">
    <slot />
  </button>
</template>
