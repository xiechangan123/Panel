import { useGettext } from 'vue3-gettext'

import website from '@/api/panel/website'
import { notifyWebsiteCreated } from '@/utils'

export interface ProxyWebsiteForm {
  name: string
  domains: string[]
  port: number | null
}

// 创建项目和管理项目反代共用：校验并创建绑定到项目的反向代理网站
export function useProxyWebsite() {
  const { $gettext } = useGettext()

  // 只做必填检查，域名冲突等由创建接口返回
  const validateProxyWebsite = (form: ProxyWebsiteForm) => {
    if (!form.domains.some((d) => d.trim() !== '')) {
      window.$message.warning($gettext('Please enter at least one domain'))
      return false
    }
    if (!form.port) {
      window.$message.warning($gettext('Please enter the project port'))
      return false
    }
    return true
  }

  const createProxyWebsite = (project: { id: number; name: string }, form: ProxyWebsiteForm) =>
    new Promise<boolean>((resolve) => {
      useRequest(
        website.create({
          type: 'proxy',
          name: form.name,
          domains: form.domains.filter((d) => d.trim() !== ''),
          listens: ['80'],
          proxy: `http://127.0.0.1:${form.port}`,
          remark: $gettext('Auto-created for project: %{ name }', { name: project.name }),
          project_id: project.id,
        }),
      )
        .onSuccess(({ data }) => {
          notifyWebsiteCreated(form.name, data.warnings)
          window.$bus.emit('website:refresh')
          resolve(true)
        })
        .onError(() => resolve(false))
    })

  return { validateProxyWebsite, createProxyWebsite }
}
