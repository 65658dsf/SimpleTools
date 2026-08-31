<script setup lang="ts">
import type { Component } from 'vue'
import type { ToolId } from '../types'

export interface ToolNavItem {
  id: ToolId
  icon: Component
  label: string
}

const props = defineProps<{
  items: ToolNavItem[]
  activePath: string
  mobile?: boolean
}>()

const emit = defineEmits<{
  navigate: [id: ToolId]
}>()

function isActive(id: ToolId) {
  return props.activePath === `/${id}`
}
</script>

<template>
  <button
    v-for="item in props.items"
    :key="item.id"
    :class="[props.mobile ? 'mobile-tool-link' : 'tool-link', { active: isActive(item.id) }]"
    @click="emit('navigate', item.id)"
  >
    <component :is="item.icon" :size="props.mobile ? 16 : 18" />
    <span>{{ item.label }}</span>
  </button>
</template>
