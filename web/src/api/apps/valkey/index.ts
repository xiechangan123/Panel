import { http } from '@/utils'

export default {
  load: (): any => http.Get('/apps/valkey/load'),
  config: (): any => http.Get('/apps/valkey/config'),
  saveConfig: (config: string): any => http.Post('/apps/valkey/config', { config }),
  configTune: (): any => http.Get('/apps/valkey/config_tune'),
  saveConfigTune: (data: any): any => http.Post('/apps/valkey/config_tune', data),
  // 慢日志
  slowLog: (): any => http.Get('/apps/valkey/slow_log'),
  resetSlowLog: (): any => http.Post('/apps/valkey/slow_log/reset'),
  // 客户端连接
  clients: (): any => http.Get('/apps/valkey/clients'),
  killClient: (id: number): any => http.Post('/apps/valkey/clients/kill', { id }),
  // 内存诊断
  memory: (): any => http.Get('/apps/valkey/memory'),
  // 扫描大 Key
  scanBigKeys: (): any => http.Post('/apps/valkey/bigkeys'),
}
