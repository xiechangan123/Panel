import { request } from '@/utils'

export default {
  // 负载状态
  load: (): any => request.get('/apps/mysql/load'),
  // 获取配置
  config: (): any => request.get('/apps/mysql/config'),
  // 保存配置
  saveConfig: (config: string): any => request.post('/apps/mysql/config', { config }),
  // 获取错误日志
  errorLog: (): any => request.get('/apps/mysql/errorLog'),
  // 清空错误日志
  clearErrorLog: (): any => request.post('/apps/mysql/clearErrorLog'),
  // 获取慢查询日志
  slowLog: (): any => request.get('/apps/mysql/slowLog'),
  // 清空慢查询日志
  clearSlowLog: (): any => request.post('/apps/mysql/clearSlowLog'),
  // 获取 root 密码
  rootPassword: (): any => request.get('/apps/mysql/rootPassword'),
  // 修改 root 密码
  setRootPassword: (password: string): any => request.post('/apps/mysql/rootPassword', { password })
}
