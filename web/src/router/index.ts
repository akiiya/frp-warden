import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { guest: true } },
    {
      path: '/',
      component: () => import('../layouts/AdminLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', name: 'dashboard', component: () => import('../views/DashboardView.vue') },
        { path: 'tenants', name: 'tenants', component: () => import('../views/TenantsView.vue') },
        { path: 'domain-zones', name: 'domain-zones', component: () => import('../views/DomainZonesView.vue') },
        { path: 'resources', name: 'resources', component: () => import('../views/ResourcesView.vue') },
        { path: 'grants', name: 'grants', component: () => import('../views/GrantsView.vue') },
        { path: 'proxies', name: 'proxies', component: () => import('../views/ProxiesView.vue') },
        { path: 'audit-logs', name: 'audit-logs', component: () => import('../views/AuditLogsView.vue') },
        { path: 'account', name: 'account', component: () => import('../views/AccountView.vue') },
      ],
    },
  ],
})

// 路由守卫：未登录跳转 /login；已登录访问 /login 跳转 /dashboard。
router.beforeEach(async (to) => {
  const auth = useAuthStore()

  // 首次加载时获取当前用户信息。
  if (auth.loading && auth.admin === null) {
    await auth.fetchMe()
  }

  if (to.meta.requiresAuth && !auth.admin) {
    return '/login'
  }
  if (to.meta.guest && auth.admin) {
    return '/dashboard'
  }
})

export default router
