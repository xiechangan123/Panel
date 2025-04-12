import { $gettext } from '@/utils/gettext'
import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'benchmark',
  path: '/apps/benchmark',
  component: Layout,
  isHidden: true,
  children: [
    {
      name: 'apps-benchmark-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: $gettext('Rat Benchmark'),
        icon: 'dashicons:performance',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
