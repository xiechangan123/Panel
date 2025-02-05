import { request } from '@/utils'

export default {
  // 负载状态
  load: (): any => request.get('/apps/postgresql/load'),
  // 获取配置
  config: (): any => request.get('/apps/postgresql/config'),
  // 保存配置
  saveConfig: (config: string): any => request.post('/apps/postgresql/config', { config }),
  // 获取用户配置
  userConfig: (): any => request.get('/apps/postgresql/userConfig'),
  // 保存配置
  saveUserConfig: (config: string): any => request.post('/apps/postgresql/userConfig', { config }),
  // 获取日志
  log: (): any => request.get('/apps/postgresql/log'),
  // 清空错误日志
  clearLog: (): any => request.post('/apps/postgresql/clearLog')
}
