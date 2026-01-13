import { http } from '@/utils'

export default {
  // 获取模版列表
  list: (): any => http.Get('/template'),
  // 获取模版详情
  get: (slug: string): any => http.Get(`/template/${slug}`),
  // 使用模版创建编排
  create: (data: {
    slug: string
    name: string
    envs: { key: string; value: string }[]
    auto_firewall: boolean
  }): any => http.Post('/template', data),
  // 模版下载回调
  callback: (slug: string): any => http.Post(`/template/${slug}/callback`)
}
