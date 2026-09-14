import { http } from '@/utils'

export default {
  // 负载状态
  load: (): any => http.Get('/apps/openlitespeed/load'),
  // 获取配置
  config: (): any => http.Get('/apps/openlitespeed/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/openlitespeed/config', { config }),
  // 获取错误日志
  errorLog: (): any => http.Get('/apps/openlitespeed/error_log'),
  // 清空错误日志
  clearErrorLog: (): any => http.Post('/apps/openlitespeed/clear_error_log'),
  // PHP 版本运行协议列表
  php: (): any => http.Get('/apps/openlitespeed/php'),
  // 切换 PHP 版本运行协议
  setPHP: (version: number, lsapi: boolean): any =>
    http.Post('/apps/openlitespeed/php', { version, lsapi }),
  // 服务器级真实 IP 配置
  realIP: (): any => http.Get('/apps/openlitespeed/realip'),
  // 保存服务器级真实 IP 配置
  setRealIP: (data: any): any => http.Post('/apps/openlitespeed/realip', data),
}
