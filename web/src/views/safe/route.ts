import type { RouteType } from '@/types/router'

const Layout = () => import('@/layouts/IndexView.vue')

export default {
  name: 'safe',
  path: '/safe',
  component: Layout,
  meta: {
    order: 40,
  },
  children: [
    {
      name: 'safe-index',
      path: '',
      component: () => import('@/views/firewall/IndexView.vue'),
      meta: {
        title: 'Security',
        icon: 'mdi:security',
        role: ['admin'],
        requireAuth: true,
      },
    },
  ],
} as RouteType
