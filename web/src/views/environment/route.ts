import type { RouteType } from '~/types/router'

const Layout = () => import('@/layout/IndexView.vue')

export default {
  name: 'environment',
  path: '/environment',
  isHidden: true,
  component: Layout,
  children: [
    {
      name: 'environment-php',
      path: 'php/:slug',
      isHidden: true,
      component: () => import('./PHPView.vue'),
      meta: {
        title: 'PHP',
        icon: 'mdi:language-php',
        role: ['admin'],
        requireAuth: true
      }
    }
  ]
} as RouteType
