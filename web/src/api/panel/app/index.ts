import { request } from '@/utils'

export default {
  // 获取应用列表
  list: (page: number, limit: number): any => request.get('/app/list', { params: { page, limit } }),
  // 安装应用
  install: (slug: string, channel: string | null): any =>
    request.post('/app/install', { slug, channel }),
  // 卸载应用
  uninstall: (slug: string): any => request.post('/app/uninstall', { slug }),
  // 更新应用
  update: (slug: string): any => request.post('/app/update', { slug }),
  // 设置首页显示
  updateShow: (slug: string, show: boolean): any => request.post('/app/updateShow', { slug, show }),
  // 应用是否已安装
  isInstalled: (slug: string): any => request.get('/app/isInstalled', { params: { slug } }),
  // 更新缓存
  updateCache: (): any => request.get('/app/updateCache')
}
