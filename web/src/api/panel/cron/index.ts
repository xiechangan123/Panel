import { http } from '@/utils'

export default {
  // 获取任务列表
  list: (page: number, limit: number): any => http.Get('/cron', { params: { page, limit } }),
  // 获取任务脚本
  get: (id: number): any => http.Get('/cron/' + id),
  // 创建任务
  create: (task: any): any => http.Post('/cron', task),
  // 修改任务
  update: (id: number, name: string, time: string, script: string): any =>
    http.Put('/cron/' + id, { name, time, script }),
  // 删除任务
  delete: (id: number): any => http.Delete(`/cron/${id}`),
  // 修改任务状态
  status: (id: number, status: boolean): any => http.Post('/cron/' + id + '/status', { status })
}
