<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Check, Languages, Monitor, Moon, Sun } from 'lucide-vue-next'
import { onMounted } from 'vue'
import { initPreferences, setThemeMode, themeMode } from '../preferences'
const { t, locale } = useI18n(); const theme = themeMode
onMounted(initPreferences)
function setTheme(value: 'light' | 'dark' | 'system') { setThemeMode(value) }
</script>
<template><section class="page-header"><div><div class="eyebrow"><Languages :size="14" /> {{ t('preferences') }}</div><h1>{{ t('preferences') }}</h1><p>{{ t('tagline') }}</p></div></section><section class="settings-page panel"><div class="settings-block"><div class="settings-title"><Sun :size="18" /><div><h2>{{ t('theme') }}</h2><p>{{ t('saved') }}</p></div></div><div class="option-grid"><button :class="{ selected: theme === 'light' }" @click="setTheme('light')"><Sun :size="17" />{{ t('light') }}<Check v-if="theme === 'light'" :size="15" /></button><button :class="{ selected: theme === 'dark' }" @click="setTheme('dark')"><Moon :size="17" />{{ t('dark') }}<Check v-if="theme === 'dark'" :size="15" /></button><button :class="{ selected: theme === 'system' }" @click="setTheme('system')"><Monitor :size="17" />{{ t('system') }}<Check v-if="theme === 'system'" :size="15" /></button></div></div><div class="settings-block"><div class="settings-title"><Languages :size="18" /><div><h2>{{ t('language') }}</h2><p>{{ t('saved') }}</p></div></div><div class="option-grid"><button :class="{ selected: locale === 'en' }" @click="locale = 'en'"><span>EN</span>English<Check v-if="locale === 'en'" :size="15" /></button><button :class="{ selected: locale === 'zh' }" @click="locale = 'zh'"><span>中</span>简体中文<Check v-if="locale === 'zh'" :size="15" /></button></div></div></section></template>
