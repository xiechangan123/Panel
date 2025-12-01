import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'project',
  path: '/project',
  component: Layout,
  meta: {
    order: 3
  },
  children: [
    {
      name: 'project-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'Projects',
        icon: 'mdi:folder-multiple',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
