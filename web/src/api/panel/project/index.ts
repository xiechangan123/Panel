import { http } from '@/utils'

export default {
  // 获取项目列表
  list: (type: string, page: number, limit: number, keyword?: string): any =>
    http.Get('/project', { params: { type, page, limit, keyword } }),
  // 获取项目详情
  get: (id: number): any => http.Get(`/project/${id}`),
  // 创建项目
  create: (data: any): any => http.Post('/project', data),
  // 更新项目
  update: (id: number, data: any): any => http.Put(`/project/${id}`, data),
  // 删除项目，websites 为 true 时同时删除绑定的网站
  delete: (id: number, websites = false): any => http.Delete(`/project/${id}`, { websites }),
}
