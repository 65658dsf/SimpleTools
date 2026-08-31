import { createRouter, createWebHashHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/convert' },
    { path: '/:tool(convert|compress|pdf)', component: () => import('./views/WorkspaceView.vue') },
    { path: '/watermark', component: () => import('./views/WatermarkView.vue') },
    { path: '/qrcode', component: () => import('./views/QRCodeView.vue') },
    { path: '/recent', component: () => import('./views/RecentView.vue') },
    { path: '/settings', component: () => import('./views/SettingsView.vue') },
  ],
})
