import type { RouteType } from '~/types/router'
import { $gettext } from '@/utils/gettext'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'fail2ban',
  path: '/apps/fail2ban',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-fail2ban-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('Fail2ban'),
        icon: 'mdi:wall-fire',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
