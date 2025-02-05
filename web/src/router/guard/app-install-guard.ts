import app from '@/api/panel/app'
import type { Router } from 'vue-router'

export function createAppInstallGuard(router: Router) {
  router.beforeEach(async (to) => {
    const slug = to.path.split('/').pop()
    if (to.path.startsWith('/apps/') && slug) {
      useRequest(app.isInstalled(slug)).onSuccess(({ data }) => {
        if (!data.installed) {
          window.$message.error(`应用 ${data.name} 未安装`)
          return router.push({ name: 'app-index' })
        }
      })
    }

    // 网站
    if (to.path.startsWith('/website')) {
      useRequest(app.isInstalled('nginx')).onSuccess(({ data }) => {
        if (!data.installed) {
          window.$message.error(`Web 服务器 ${data.name} 未安装`)
          return router.push({ name: 'app-index' })
        }
      })
    }

    // 容器
    if (to.path.startsWith('/container')) {
      useRequest(app.isInstalled('docker')).onSuccess(({ data }) => {
        if (!data.installed) {
          useRequest(app.isInstalled('podman')).onSuccess(({ data }) => {
            if (!data.installed) {
              window.$message.error(`容器引擎 Docker / Podman 未安装`)
              return router.push({ name: 'app-index' })
            }
          })
        }
      })
    }
  })
}
