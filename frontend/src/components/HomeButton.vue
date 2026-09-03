<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const props = defineProps<{ disabled?: boolean }>()

function preventNavigation(event: MouseEvent) {
  if (props.disabled) event.preventDefault()
}
</script>

<template>
  <RouterLink
    to="/"
    class="home-button"
    :class="{ disabled: props.disabled }"
    :aria-label="t('backHome')"
    :aria-disabled="props.disabled || undefined"
    :tabindex="props.disabled ? -1 : undefined"
    @click="preventNavigation"
  >
    <ArrowLeft :size="15" aria-hidden="true" />
    <span>{{ t('backHome') }}</span>
  </RouterLink>
</template>
