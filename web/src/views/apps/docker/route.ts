import type { RouteType } from '~/types/router'
import { $gettext } from '@/utils/gettext'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'docker',
  path: '/apps/docker',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-docker-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('Docker'),
        icon: 'logos:docker-icon',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
