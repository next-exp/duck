import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import DetectorView from '../views/DetectorView.vue'
import TestDevicesView from '../views/TestDevicesView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/detector', name: 'detector', component: DetectorView },
    { path: '/test-devices', name: 'test-devices', component: TestDevicesView },
  ]
})

export default router
