import type { RouteType } from '@/types/router'

const Layout = () => import('@/layouts/IndexView.vue')

export default {
  name: 'tamper',
  path: '/tamper',
  component: Layout,
  meta: {
    order: 41,
  },
  children: [
    {
      name: 'tamper-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'Tamper Protection',
        icon: 'mdi:shield-lock-outline',
        role: ['admin'],
        requireAuth: true,
      },
    },
  ],
} as RouteType
