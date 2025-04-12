import type { Router } from 'vue-router'

import { useThemeStore } from '@/store'
import { translateTitle } from '@/utils'

export function createPageTitleGuard(router: Router) {
  const themeStore = useThemeStore()
  router.afterEach((to) => {
    const pageTitle = translateTitle(String(to.meta.title))
    if (pageTitle) document.title = `${pageTitle} | ${themeStore.name}`
    else document.title = themeStore.name
  })
}
