import { http } from '@/utils'

export default {
  // 获取信息
  info: (): any => http.Get('/apps/phpmyadmin/info'),
  // 以指定 MySQL 服务器的凭据登录并下发会话 Cookie
  login: (serverID: number): any => http.Post('/apps/phpmyadmin/login', { server_id: serverID }),
  // 设置端口
  port: (port: number): any => http.Post('/apps/phpmyadmin/port', { port }),
  // 获取配置
  config: (): any => http.Get('/apps/phpmyadmin/config'),
  // 保存配置
  updateConfig: (config: string): any => http.Post('/apps/phpmyadmin/config', { config }),
}
