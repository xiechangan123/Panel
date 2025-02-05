import { request } from '@/utils'

export default {
  // 获取信息
  info: (): any => request.get('/apps/phpmyadmin/info'),
  // 设置端口
  port: (port: number): any => request.post('/apps/phpmyadmin/port', { port }),
  // 获取配置
  getConfig: (): any => request.get('/apps/phpmyadmin/config'),
  // 保存配置
  saveConfig: (config: string): any => request.post('/apps/phpmyadmin/config', { config })
}
