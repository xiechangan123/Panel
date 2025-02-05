import { request } from '@/utils'

export default {
  // 负载状态
  load: (): any => request.get('/apps/nginx/load'),
  // 获取配置
  config: (): any => request.get('/apps/nginx/config'),
  // 保存配置
  saveConfig: (config: string): any => request.post('/apps/nginx/config', { config }),
  // 获取错误日志
  errorLog: (): any => request.get('/apps/nginx/errorLog'),
  // 清空错误日志
  clearErrorLog: (): any => request.post('/apps/nginx/clearErrorLog')
}
