import { http, request } from '@/utils'

export default {
  // 列表
  list: (page: number, limit: number): any => request.get('/website', { params: { page, limit } }),
  // 创建
  create: (data: any): any => request.post('/website', data),
  // 删除
  delete: (id: number, path: boolean, db: boolean): any =>
    request.delete(`/website/${id}`, { data: { path, db } }),
  // 伪静态
  rewrites: () => http.Get(`/website/rewrites`),
  // 获取默认配置
  defaultConfig: (): any => request.get('/website/defaultConfig'),
  // 保存默认配置
  saveDefaultConfig: (index: string, stop: string): any =>
    request.post('/website/defaultConfig', { index, stop }),
  // 网站配置
  config: (id: number): any => request.get('/website/' + id),
  // 保存网站配置
  saveConfig: (id: number, data: any): any => request.put(`/website/${id}`, data),
  // 清空日志
  clearLog: (id: number): any => request.delete('/website/' + id + '/log'),
  // 更新备注
  updateRemark: (id: number, remark: string): any =>
    request.post(`/website/${id}` + '/updateRemark', { remark }),
  // 重置配置
  resetConfig: (id: number): any => request.post(`/website/${id}/resetConfig`),
  // 修改状态
  status: (id: number, status: boolean): any => request.post(`/website/${id}/status`, { status }),
  // 签发证书
  obtainCert: (id: number): any => request.post(`/website/${id}/obtainCert`)
}
