import { request } from '@/utils'

export default {
  // 获取任务列表
  list: (page: number, limit: number): any => request.get('/cron', { params: { page, limit } }),
  // 获取任务脚本
  get: (id: number): any => request.get('/cron/' + id),
  // 创建任务
  create: (task: any): any => request.post('/cron', task),
  // 修改任务
  update: (id: number, name: string, time: string, script: string): any =>
    request.put('/cron/' + id, { name, time, script }),
  // 删除任务
  delete: (id: number): any => request.delete('/cron/' + id),
  // 修改任务状态
  status: (id: number, status: boolean): any => request.post('/cron/' + id + '/status', { status })
}
