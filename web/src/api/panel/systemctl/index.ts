import { http } from '@/utils'

export default {
  // 服务状态
  status: (service: string): any => http.Get('/systemctl/status', { params: { service } }),
  // 是否启用服务
  isEnabled: (service: string): any => http.Get('/systemctl/isEnabled', { params: { service } }),
  // 启用服务
  enable: (service: string): any => http.Post('/systemctl/enable', { service }),
  // 禁用服务
  disable: (service: string): any => http.Post('/systemctl/disable', { service }),
  // 重启服务
  restart: (service: string): any => http.Post('/systemctl/restart', { service }),
  // 重载服务
  reload: (service: string): any => http.Post('/systemctl/reload', { service }),
  // 启动服务
  start: (service: string): any => http.Post('/systemctl/start', { service }),
  // 停止服务
  stop: (service: string): any => http.Post('/systemctl/stop', { service })
}
