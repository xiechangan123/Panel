import { request } from '@/utils'

export default {
  // 获取配置
  config: (): any => request.get('/apps/gitea/config'),
  // 保存配置
  saveConfig: (config: string): any => request.post('/apps/gitea/config', { config })
}
