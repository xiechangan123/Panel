import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'dashboard',
  path: '/',
  component: Layout,
  redirect: '/dashboard',
  meta: {
    order: 0
  },
  children: [
    {
      name: 'dashboard-index',
      path: 'dashboard',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'Dashboard',
        icon: 'mdi:gauge',
        role: ['admin'],
        requireAuth: true
      }
    },
    {
      name: 'dashboard-update',
      path: 'update',
      component: () => import('./UpdateView.vue'),
      isHidden: true,
      meta: {
        title: 'Update',
        icon: 'mdi:archive-arrow-up-outline',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
