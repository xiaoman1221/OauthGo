import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue')
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('../views/Register.vue')
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('../views/ForgotPassword.vue')
  },
  {
    path: '/oauth-callback',
    name: 'OauthCallback',
    component: () => import('../views/OauthCallback.vue')
  },
  {
    path: '/',
    component: () => import('../layout/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
      { path: 'apps', name: 'Apps', component: () => import('../views/Apps.vue') },
      { path: 'logins', name: 'Logins', component: () => import('../views/Logins.vue') },
      {
        path: 'notifications',
        name: 'Notifications',
        component: () => import('../views/Notifications.vue')
      },
      { path: 'providers', name: 'Providers', component: () => import('../views/Providers.vue') },
      { path: 'docs/providers', name: 'ProviderDocs', component: () => import('../views/ProviderDocs.vue') },
      { path: 'docs/service', name: 'ServiceDocs', component: () => import('../views/ServiceDocs.vue') },
      { path: 'user-center', name: 'UserCenter', component: () => import('../views/UserCenter.vue') },
      { path: 'settings', name: 'Settings', component: () => import('../views/Settings.vue') },
      { path: 'users', name: 'Users', component: () => import('../views/Users.vue') }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const userStore = useUserStore()
  const publicPaths = ['/login', '/register', '/forgot-password', '/oauth-callback']
  if (!publicPaths.includes(to.path) && !userStore.isLogin) {
    return '/login'
  }
  if (to.path === '/login' && userStore.isLogin) {
    return '/'
  }
  return true
})

export default router
