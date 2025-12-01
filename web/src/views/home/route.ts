import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'home',
  path: '/',
  component: Layout,
  meta: {
    order: 0
  },
  children: [
    {
      name: 'home-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'Home',
        icon: 'mdi:house-outline',
        role: ['admin'],
        requireAuth: true
      }
    },
    {
      name: 'home-update',
      path: 'update',
      component: () => import('./UpdateView.vue'),
      isHidden: true,
      meta: {
        title: 'Update',
        icon: 'mdi:archive-arrow-up-outline',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
