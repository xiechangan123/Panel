import { http } from '@/utils'

export default {
  // 获取配置
  config: (name: string): any => http.Get('/apps/frp/config', { params: { name } }),
  // 保存配置
  saveConfig: (name: string, config: string): any =>
    http.Post('/apps/frp/config', { name, config }),
  // 获取运行用户
  user: (name: string): any => http.Get('/apps/frp/user', { params: { name } }),
  // 设置运行用户
  saveUser: (name: string, user: string, group: string): any =>
    http.Post('/apps/frp/user', { name, user, group }),
  // 获取 frps 参数
  server: (): any => http.Get('/apps/frp/server'),
  // 保存 frps 参数
  saveServer: (tune: any): any => http.Post('/apps/frp/server', tune),
  // 获取 frpc 公共参数
  client: (): any => http.Get('/apps/frp/client'),
  // 保存 frpc 公共参数
  saveClient: (tune: any): any => http.Post('/apps/frp/client', tune),
  // 代理列表
  proxies: (page: number, limit: number): any =>
    http.Get('/apps/frp/proxies', { params: { page, limit } }),
  // 添加代理
  addProxy: (proxy: any): any => http.Post('/apps/frp/proxies', proxy),
  // 更新代理
  updateProxy: (name: string, proxy: any): any => http.Post(`/apps/frp/proxies/${name}`, proxy),
  // 删除代理
  deleteProxy: (name: string): any => http.Delete(`/apps/frp/proxies/${name}`),
  // 访问者列表
  visitors: (page: number, limit: number): any =>
    http.Get('/apps/frp/visitors', { params: { page, limit } }),
  // 添加访问者
  addVisitor: (visitor: any): any => http.Post('/apps/frp/visitors', visitor),
  // 更新访问者
  updateVisitor: (name: string, visitor: any): any =>
    http.Post(`/apps/frp/visitors/${name}`, visitor),
  // 删除访问者
  deleteVisitor: (name: string): any => http.Delete(`/apps/frp/visitors/${name}`),
}
