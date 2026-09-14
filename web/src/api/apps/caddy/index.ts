import { http } from '@/utils'

export default {
  // 获取配置
  config: (): any => http.Get('/apps/caddy/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/caddy/config', { config }),
  // 获取错误日志
  errorLog: (): any => http.Get('/apps/caddy/error_log'),
  // 清空错误日志
  clearErrorLog: (): any => http.Post('/apps/caddy/clear_error_log'),
}
