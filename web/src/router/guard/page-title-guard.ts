import type { Router } from 'vue-router'

import { translateTitle } from '@/locales/menu'
import { useThemeStore } from '@/store'

export function createPageTitleGuard(router: Router) {
  const themeStore = useThemeStore()
  router.afterEach((to) => {
    const pageTitle = typeof to.meta.title === 'string' ? translateTitle(to.meta.title) : '404'
    if (pageTitle) document.title = `${pageTitle} | ${themeStore.name}`
    else document.title = themeStore.name
  })
}
