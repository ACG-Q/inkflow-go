import { createRouter, createWebHistory } from 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    hideNav?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/Home.vue') },
    { path: '/sign', name: 'sign', component: () => import('@/views/SignFront.vue') },
    { path: '/admin/login', name: 'login', component: () => import('@/views/AdminLogin.vue'), meta: { hideNav: true } },
    { path: '/admin', name: 'dashboard', component: () => import('@/views/AdminDashboard.vue'), meta: { requiresAuth: true } },
    { path: '/admin/editor', name: 'editor', component: () => import('@/views/AdminEditor.vue'), meta: { requiresAuth: true } },
    { path: '/admin/records', name: 'records', component: () => import('@/views/AdminRecords.vue'), meta: { requiresAuth: true } },
    { path: '/admin/fonts', name: 'fonts', component: () => import('@/views/AdminFonts.vue'), meta: { requiresAuth: true } },
    { path: '/admin/settings', name: 'settings', component: () => import('@/views/AdminSettings.vue'), meta: { requiresAuth: true } },
  ],
})

router.beforeEach((to, _from, next) => {
  if (to.meta?.requiresAuth) {
    const token = localStorage.getItem('token')
    if (!token) {
      next({ name: 'login' })
      return
    }
    try {
      const payload = JSON.parse(atob(token.split('.')[1]))
      if (payload.exp && payload.exp * 1000 < Date.now()) {
        localStorage.removeItem('token')
        next({ name: 'login' })
        return
      }
    } catch {
      localStorage.removeItem('token')
      next({ name: 'login' })
      return
    }
  }
  next()
})

export default router
