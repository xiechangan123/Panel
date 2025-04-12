import type { RouteType } from '~/types/router'
import { $gettext } from '@/utils/gettext'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'monitor',
  path: '/monitor',
  component: Layout,
  meta: {
    order: 20
  },
  children: [
    {
      name: 'monitor-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('Monitoring'),
        icon: 'mdi:chart-line',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
