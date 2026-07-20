import { http } from '@/utils'

export default {
  // 获取信息
  info: (): any => http.Get('/apps/pgadmin/info'),
  // 设置端口
  port: (port: number): any => http.Post('/apps/pgadmin/port', { port }),
  // 同步面板全部 PostgreSQL 服务器并登录,下发会话 Cookie
  login: (): any => http.Post('/apps/pgadmin/login'),
  // 重置管理员密码
  resetPassword: (password: string): any => http.Post('/apps/pgadmin/reset_password', { password }),
}
