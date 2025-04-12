import type { RouteType } from '~/types/router'

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
        title: 'Website',
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
        title: 'Website Status',
        icon: 'mdi:web',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
