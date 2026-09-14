import type { RouteType } from '@/types/router'

const Layout = () => import('@/layouts/IndexView.vue')

export default {
  name: 'openlitespeed',
  path: '/apps/openlitespeed',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-openlitespeed-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'OpenLiteSpeed',
        role: ['admin'],
        requireAuth: true,
      },
    },
  ],
} as RouteType
