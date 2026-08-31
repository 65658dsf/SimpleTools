<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  AlertCircle,
  CheckCircle2,
  Clock3,
  FileImage,
  FileText,
  FolderOpen,
  Loader2,
  Play,
  QrCode,
  Stamp,
  Trash2,
} from 'lucide-vue-next'
import { useWorkspaceStore } from '../stores/workspace'
import { wailsService } from '../services/wails'
import type { RecentJobSummary, ToolId } from '../types'

const { t, locale } = useI18n()
const router = useRouter()
const store = useWorkspaceStore()
const busyJobId = ref('')
const messages = ref<Record<string, string>>({})

const dateFormatter = computed(
  () =>
    new Intl.DateTimeFormat(locale.value, {
      dateStyle: 'medium',
      timeStyle: 'short',
    }),
)

function toolIcon(tool: ToolId) {
  if (tool === 'pdf') return FileText
  if (tool === 'watermark') return Stamp
  if (tool === 'qrcode') return QrCode
  return FileImage
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : dateFormatter.value.format(date)
}

function countSummary(job: RecentJobSummary) {
  const parts = [`${job.total} ${t('recentFiles')}`, `${job.completed} ${t('recentCompleted')}`]
  if (job.failed > 0) parts.push(`${job.failed} ${t('recentFailed')}`)
  if ((job.cancelled ?? 0) > 0) parts.push(`${job.cancelled} ${t('recentCancelled')}`)
  return parts.join(' · ')
}

function canRerun(job: RecentJobSummary) {
  return wailsService.isNative() && Boolean(job.request?.inputs?.length || job.inputPaths?.length)
}

function canOpenOutput(job: RecentJobSummary) {
  return (
    wailsService.isNative() &&
    Boolean((job.outputDirectory || job.request?.outputDirectory || '').trim())
  )
}

function setMessage(jobId: string, message: string) {
  messages.value = { ...messages.value, [jobId]: message }
}

async function openOutput(job: RecentJobSummary) {
  if (!canOpenOutput(job)) {
    setMessage(job.id, wailsService.isNative() ? t('recentNoOutput') : t('recentOutputUnavailable'))
    return
  }
  busyJobId.value = job.id
  const result = await store.openRecentOutputDirectory(job)
  busyJobId.value = ''
  if (!result.ok)
    setMessage(
      job.id,
      result.message === 'desktop-only' ? t('recentOutputUnavailable') : t('recentNoOutput'),
    )
}

async function rerun(job: RecentJobSummary) {
  if (!canRerun(job)) {
    setMessage(job.id, t('recentRerunUnavailable'))
    return
  }
  busyJobId.value = job.id
  const result = await store.rerunRecentJob(job)
  busyJobId.value = ''
  if (!result.ok) {
    setMessage(
      job.id,
      result.message === 'processing' ? t('recentProcessing') : t('recentRerunMissing'),
    )
    return
  }
  await router.push(`/${job.tool}`)
}

function remove(job: RecentJobSummary) {
  store.removeRecentJob(job.id)
  const next = { ...messages.value }
  delete next[job.id]
  messages.value = next
}
</script>

<template>
  <section class="page-header recent-page-header">
    <div>
      <div class="eyebrow"><Clock3 :size="14" /> {{ t('appName') }}</div>
      <h1>{{ t('recent') }}</h1>
      <p>{{ t('recentEmptyHint') }}</p>
    </div>
    <div class="header-stat">
      <span class="stat-value">{{ store.recentJobs.length }}</span>
      <span class="stat-label">{{ t('recent') }}</span>
    </div>
  </section>

  <section v-if="store.recentJobs.length" class="recent-list" aria-live="polite">
    <article v-for="job in store.recentJobs" :key="job.id" class="recent-job panel">
      <div class="recent-job-main">
        <div class="recent-job-icon"><component :is="toolIcon(job.tool)" :size="19" /></div>
        <div class="recent-job-content">
          <div class="recent-job-title-row">
            <h2>{{ t(job.tool) }}</h2>
            <time :datetime="job.finishedAt">{{ formatDate(job.finishedAt) }}</time>
          </div>
          <p class="recent-job-counts">
            <CheckCircle2 :size="14" />
            <span>{{ countSummary(job) }}</span>
          </p>
          <p
            class="recent-job-output"
            :title="job.outputDirectory || job.request?.outputDirectory || t('recentNoOutput')"
          >
            <FolderOpen :size="14" />
            <span>{{
              job.outputDirectory || job.request?.outputDirectory || t('recentNoOutput')
            }}</span>
          </p>
          <p v-if="!canRerun(job) && !messages[job.id]" class="recent-job-message" role="status">
            <AlertCircle :size="14" /> {{ t('recentRerunUnavailable') }}
          </p>
          <p v-if="messages[job.id]" class="recent-job-message" role="status">
            <AlertCircle :size="14" /> {{ messages[job.id] }}
          </p>
        </div>
      </div>
      <div class="recent-job-actions">
        <button
          class="secondary-button"
          :disabled="busyJobId === job.id"
          :title="t('recentOpenFolder')"
          @click="openOutput(job)"
        >
          <Loader2 v-if="busyJobId === job.id" class="spin" :size="15" />
          <FolderOpen v-else :size="15" />
          {{ t('recentOpenFolder') }}
        </button>
        <button
          class="secondary-button"
          :disabled="busyJobId === job.id || !canRerun(job)"
          :title="t('recentRerun')"
          @click="rerun(job)"
        >
          <Play :size="15" /> {{ t('recentRerun') }}
        </button>
        <button
          class="icon-button small"
          :title="t('recentDelete')"
          :aria-label="t('recentDelete')"
          @click="remove(job)"
        >
          <Trash2 :size="15" />
        </button>
      </div>
    </article>
  </section>

  <section v-else class="panel empty-state recent-empty-state">
    <div class="empty-icon"><Clock3 :size="20" /></div>
    <strong>{{ t('recentEmpty') }}</strong>
    <span>{{ t('recentEmptyHint') }}</span>
  </section>
</template>
