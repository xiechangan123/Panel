import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'postgresql16',
  path: '/apps/postgresql16',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-postgresql16-index',
      path: '',
      component: () => import('../postgresql/IndexView.vue'),
      meta: {
        title: 'PostgreSQL 16',
        icon: 'mdi:database',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
