import type { Router } from 'vue-router'

import user from '@/api/panel/user'
import { useUserStore } from '@/stores'

let verified = false
let verifying: Promise<boolean> | null = null

async function verifyLogin() {
  const userStore = useUserStore()
  if (verified && userStore.id) {
    return true
  }
  if (verifying) {
    return verifying
  }

  verifying = (async () => {
    const loggedIn = await user.isLogin().send(true)
    if (!loggedIn) {
      userStore.$reset()
      return false
    }

    const info = await user.info({ meta: { noAlert: true } }).send(true)
    userStore.set(info)
    verified = true
    return true
  })()

  try {
    return await verifying
  } finally {
    verifying = null
  }
}

export function createAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    if (!to.matched.some((route) => route.meta?.requireAuth)) {
      return true
    }

    try {
      if (await verifyLogin()) {
        return true
      }
    } catch {
      verified = false
    }

    return {
      path: '/login',
      query: {
        redirect: to.fullPath,
      },
    }
  })
}
