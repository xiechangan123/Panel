import { request } from '@/utils'

export default {
  // 开关
  setting: (): any => request.get('/monitor/setting'),
  // 保存天数
  updateSetting: (enabled: boolean, days: number): any =>
    request.post('/monitor/setting', { enabled, days }),
  // 清空监控记录
  clear: (): any => request.post('/monitor/clear'),
  // 监控记录
  list: (start: number, end: number): any =>
    request.get('/monitor/list', { params: { start, end } })
}
