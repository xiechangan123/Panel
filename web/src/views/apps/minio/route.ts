import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'minio',
  path: '/apps/minio',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-minio-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'Minio',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
