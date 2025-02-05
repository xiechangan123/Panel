import { request } from '@/utils'

export default {
  // 列表
  list: (page: number, limit: number): any =>
    request.get('/apps/s3fs/mounts', { params: { page, limit } }),
  // 添加
  add: (data: any): any => request.post('/apps/s3fs/mounts', data),
  // 删除
  delete: (id: number): any => request.delete('/apps/s3fs/mounts', { data: { id } })
}
