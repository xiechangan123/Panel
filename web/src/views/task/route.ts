import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'task',
  path: '/task',
  component: Layout,
  meta: {
    order: 80
  },
  children: [
    {
      name: 'task-index',
      path: '',
      component: () => import('./IndexView.vue'),
      meta: {
        title: 'Tasks',
        icon: 'mdi:timetable',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
