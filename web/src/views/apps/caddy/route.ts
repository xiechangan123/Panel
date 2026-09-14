import type { RouteType } from '@/types/router'

const Layout = () => import('@/layouts/IndexView.vue')

export default {
  name: 'caddy',
  path: '/apps/caddy',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-caddy-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'Caddy',
        role: ['admin'],
        requireAuth: true,
      },
    },
  ],
} as RouteType
