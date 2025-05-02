import { http } from '@/utils'

export default {
  // 设为 CLI 版本
  setCli: (version: number): any => http.Post(`/apps/php${version}/set_cli`),
  // 获取配置
  config: (version: number): any => http.Get(`/apps/php${version}/config`),
  // 保存配置
  saveConfig: (version: number, config: string): any =>
    http.Post(`/apps/php${version}/config`, { config }),
  // 获取FPM配置
  fpmConfig: (version: number): any => http.Get(`/apps/php${version}/fpm_config`),
  // 保存FPM配置
  saveFPMConfig: (version: number, config: string): any =>
    http.Post(`/apps/php${version}/fpm_config`, { config }),
  // 负载状态
  load: (version: number): any => http.Get(`/apps/php${version}/load`),
  // 获取错误日志
  errorLog: (version: number): any => http.Get(`/apps/php${version}/error_log`),
  // 清空错误日志
  clearErrorLog: (version: number): any => http.Post(`/apps/php${version}/clear_error_log`),
  // 获取慢日志
  slowLog: (version: number): any => http.Get(`/apps/php${version}/slow_log`),
  // 清空慢日志
  clearSlowLog: (version: number): any => http.Post(`/apps/php${version}/clear_slow_log`),
  // 拓展列表
  extensions: (version: number): any => http.Get(`/apps/php${version}/extensions`),
  // 安装拓展
  installExtension: (version: number, slug: string): any =>
    http.Post(`/apps/php${version}/extensions`, { slug }),
  // 卸载拓展
  uninstallExtension: (version: number, slug: string): any =>
    http.Delete(`/apps/php${version}/extensions`, { slug })
}
