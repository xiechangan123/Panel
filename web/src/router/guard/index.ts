import type { Router } from 'vue-router'

import { createAuthGuard } from '@/router/guard/auth-guard'
import { createTabGuard } from '@/router/guard/tab-guard'

import { createAppInstallGuard } from './app-install-guard'
import { createPageLoadingGuard } from './page-loading-guard'
import { createPageTitleGuard } from './page-title-guard'

export function setupRouterGuard(router: Router) {
  createPageLoadingGuard(router)
  createPageTitleGuard(router)
  createAuthGuard(router)
  createTabGuard(router)
  createAppInstallGuard(router)
}
