import type { RouteType } from '@/types/router'

const Layout = () => import('@/layouts/IndexView.vue')

export default {
  name: 'safe',
  path: '/safe',
  component: Layout,
  meta: {
    order: 40,
    title: 'Security',
    icon: 'mdi:security',
  },
  children: [
    {
      name: 'firewall-index',
      path: 'firewall',
      component: () => import('@/views/firewall/IndexView.vue'),
      meta: {
        title: 'Firewall',
        icon: 'mdi:firewall',
        role: ['admin'],
        requireAuth: true,
      },
    },
    {
      name: 'tamper-index',
      path: 'tamper',
      component: () => import('@/views/tamper/IndexView.vue'),
      meta: {
        title: 'Tamper Protection',
        icon: 'mdi:shield-lock-outline',
        role: ['admin'],
        requireAuth: true,
      },
    },
  ],
} as RouteType
