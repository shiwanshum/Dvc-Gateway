import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from './views/Dashboard.vue'
import PlcManager from './views/PlcManager.vue'
import TagManager from './views/TagManager.vue'

const routes = [
  { path: '/', component: Dashboard, name: 'Dashboard' },
  { path: '/plcs', component: PlcManager, name: 'PLC Management' },
  { path: '/tags', component: TagManager, name: 'Tag Management' }
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})
