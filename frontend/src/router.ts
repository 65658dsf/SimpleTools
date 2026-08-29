import { createRouter, createWebHashHistory } from 'vue-router'
import WorkspaceView from './views/WorkspaceView.vue'
import WatermarkView from './views/WatermarkView.vue'
import SettingsView from './views/SettingsView.vue'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/convert' },
    { path: '/:tool(convert|compress|pdf)', component: WorkspaceView },
    { path: '/watermark', component: WatermarkView },
    { path: '/settings', component: SettingsView },
  ],
})
