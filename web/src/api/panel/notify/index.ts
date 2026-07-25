import { http } from '@/utils'

export default {
  // 通知渠道列表
  channels: (page: number, limit: number): any =>
    http.Get('/notify/channel', { params: { page, limit } }),
  // 全部通知渠道
  allChannels: (): any => http.Get('/notify/channel/all'),
  // 新增通知渠道
  createChannel: (data: any): any => http.Post('/notify/channel', data),
  // 更新通知渠道
  updateChannel: (id: number, data: any): any => http.Put(`/notify/channel/${id}`, data),
  // 删除通知渠道
  deleteChannel: (id: number): any => http.Delete(`/notify/channel/${id}`),
  // 测试通知渠道
  testChannel: (id: number): any => http.Post(`/notify/channel/${id}/test`),
  // 事件通知设置
  setting: (): any => http.Get('/notify/setting'),
  // 保存事件通知设置
  updateSetting: (data: any): any => http.Post('/notify/setting', data),
}
