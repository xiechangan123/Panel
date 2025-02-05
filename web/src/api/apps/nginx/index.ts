import { http } from '@/utils'

export default {
  // 负载状态
  load: (): any => http.Get('/apps/nginx/load'),
  // 获取配置
  config: (): any => http.Get('/apps/nginx/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/nginx/config', { config }),
  // 获取错误日志
  errorLog: (): any => http.Get('/apps/nginx/errorLog'),
  // 清空错误日志
  clearErrorLog: (): any => http.Post('/apps/nginx/clearErrorLog')
}
