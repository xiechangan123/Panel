import type { RouteType } from '~/types/router'
import { $gettext } from '@/utils/gettext'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'firewall',
  path: '/firewall',
  component: Layout,
  meta: {
    order: 30
  },
  children: [
    {
      name: 'firewall-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('Firewall'),
        icon: 'mdi:firewall',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
