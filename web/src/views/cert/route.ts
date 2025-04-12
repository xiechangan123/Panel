import { $gettext } from '@/utils/gettext'
import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'cert',
  path: '/cert',
  component: Layout,
  meta: {
    order: 10
  },
  children: [
    {
      name: 'cert-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('Certificate'),
        icon: 'mdi:certificate-outline',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
