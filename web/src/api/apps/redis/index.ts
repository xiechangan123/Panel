import { http } from '@/utils'

export default {
  // 负载状态
  load: (): any => http.Get('/apps/redis/load'),
  // 获取配置
  config: (): any => http.Get('/apps/redis/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/redis/config', { config })
}
