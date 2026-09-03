<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, FileImage, FileOutput, FileText, QrCode, Stamp } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { ToolId } from '../types'
import { useWorkspaceStore } from '../stores/workspace'

const { t } = useI18n()
const router = useRouter()
const store = useWorkspaceStore()
const tools = computed(() => [
  { id: 'convert' as ToolId, icon: FileOutput, title: t('convert'), detail: t('convertDesc') },
  { id: 'compress' as ToolId, icon: FileImage, title: t('compress'), detail: t('compressDesc') },
  { id: 'watermark' as ToolId, icon: Stamp, title: t('watermark'), detail: t('watermarkDesc') },
  { id: 'qrcode' as ToolId, icon: QrCode, title: t('qrcode'), detail: t('qrcodeDesc') },
  { id: 'pdf' as ToolId, icon: FileText, title: t('pdf'), detail: t('pdfDesc') },
])

function openTool(tool: ToolId) {
  if (!store.setTool(tool)) return
  void router.push(`/${tool}`)
}

function canOpenTool(tool: ToolId) {
  return !store.running || tool === store.activeTool
}
</script>

<template>
  <section class="page-header home-page-header">
    <div>
      <div class="eyebrow">{{ t('appName') }}</div>
      <h1>{{ t('toolsOverview') }}</h1>
      <p>{{ t('tagline') }}</p>
    </div>
  </section>
  <section class="home-tool-grid" :aria-label="t('toolsOverview')">
    <article v-for="tool in tools" :key="tool.id" class="home-tool-card panel">
      <div class="home-tool-icon">
        <component :is="tool.icon" :size="21" />
      </div>
      <div class="home-tool-copy">
        <h2>{{ tool.title }}</h2>
        <p>{{ tool.detail }}</p>
        <button class="secondary-button home-tool-button" :disabled="!canOpenTool(tool.id)" @click="openTool(tool.id)"><span>{{ t('openTool') }}</span>
          <ArrowRight :size="15" />
        </button>
      </div>
    </article>
  </section>
</template>
