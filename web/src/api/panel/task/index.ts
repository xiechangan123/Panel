import { request } from '@/utils'

export default {
  // 获取状态
  status: (): any => request.get('/task/status'),
  // 获取任务列表
  list: (page: number, limit: number): any => request.get('/task', { params: { page, limit } }),
  // 获取任务
  get: (id: number): any => request.get('/task/' + id),
  // 删除任务
  delete: (id: number): any => request.delete('/task/' + id)
}
