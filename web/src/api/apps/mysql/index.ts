import { http } from '@/utils'

export default {
  // 负载状态
  load: (): any => http.Get('/apps/mysql/load'),
  // 获取配置
  config: (): any => http.Get('/apps/mysql/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/mysql/config', { config }),
  // 清空错误日志
  clearErrorLog: (): any => http.Post('/apps/mysql/clearErrorLog'),
  // 获取慢查询日志
  slowLog: (): any => http.Get('/apps/mysql/slowLog'),
  // 清空慢查询日志
  clearSlowLog: (): any => http.Post('/apps/mysql/clearSlowLog'),
  // 获取 root 密码
  rootPassword: (): any => http.Get('/apps/mysql/rootPassword'),
  // 修改 root 密码
  setRootPassword: (password: string): any => http.Post('/apps/mysql/rootPassword', { password })
}
