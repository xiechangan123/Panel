import type { RouteType } from '@/types/router'

const Layout = () => import('@/layouts/IndexView.vue')

export default {
  name: 'pgadmin',
  path: '/apps/pgadmin',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-pgadmin-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'pgAdmin',
        role: ['admin'],
        requireAuth: true,
      },
    },
  ],
} as RouteType
