import { http } from '@/utils'

export default {
  // 负载状态
  load: (): any => http.Get('/apps/redis/load'),
  // 获取配置
  config: (): any => http.Get('/apps/redis/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/redis/config', { config }),
  // 获取配置调整参数
  configTune: (): any => http.Get('/apps/redis/config_tune'),
  // 保存配置调整参数
  saveConfigTune: (data: any): any => http.Post('/apps/redis/config_tune', data),
  // 慢日志
  slowLog: (): any => http.Get('/apps/redis/slow_log'),
  resetSlowLog: (): any => http.Post('/apps/redis/slow_log/reset'),
  // 客户端连接
  clients: (): any => http.Get('/apps/redis/clients'),
  killClient: (id: number): any => http.Post('/apps/redis/clients/kill', { id }),
  // 内存诊断
  memory: (): any => http.Get('/apps/redis/memory'),
  // 扫描大 Key
  scanBigKeys: (): any => http.Post('/apps/redis/bigkeys'),
}
