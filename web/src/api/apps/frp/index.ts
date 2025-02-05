import { http } from '@/utils'

export default {
  // 获取配置
  config: (name: string): any => http.Get('/apps/frp/config', { params: { name } }),
  // 保存配置
  saveConfig: (name: string, config: string): any => http.Post('/apps/frp/config', { name, config })
}
