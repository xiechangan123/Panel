import type { RouteType } from '~/types/router'
import { $gettext } from '@/utils/gettext'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'website',
  path: '/website',
  component: Layout,
  meta: {
    order: 1
  },
  children: [
    {
      name: 'website-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('Websites'),
        icon: 'mdi:web',
        role: ['admin'],
        requireAuth: true
      }
    },
    {
      name: 'website-edit',
      path: 'edit/:id',
      component: () => import('./EditView.vue'),
      isHidden: true,
      meta: {
        title: $gettext('Edit Website'),
        icon: 'mdi:web',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
