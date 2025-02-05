import { request } from '@/utils'

export default {
  // 负载状态
  load: (): any => request.get('/apps/redis/load'),
  // 获取配置
  config: (): any => request.get('/apps/redis/config'),
  // 保存配置
  saveConfig: (config: string): any => request.post('/apps/redis/config', { config })
}
