import type { RouteType } from '~/types/router'
import { $gettext } from '@/utils/gettext'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'postgresql',
  path: '/apps/postgresql',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-postgresql-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('PostgreSQL'),
        icon: 'logos:postgresql',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
