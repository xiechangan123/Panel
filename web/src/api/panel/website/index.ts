import { http } from '@/utils'

export default {
  // 列表
  list: (type: string, page: number, limit: number): any =>
    http.Get('/website', { params: { type, page, limit } }),
  // 创建
  create: (data: any): any => http.Post('/website', data),
  // 删除
  delete: (id: number, path: boolean, db: boolean): any =>
    http.Delete(`/website/${id}`, { path, db }),
  // 伪静态
  rewrites: (): any => http.Get(`/website/rewrites`),
  // 获取默认配置
  defaultConfig: (): any => http.Get('/website/default_config'),
  // 保存默认配置
  saveDefaultConfig: (index: string, stop: string): any =>
    http.Post('/website/default_config', { index, stop }),
  // 网站配置
  config: (id: number): any => http.Get('/website/' + id),
  // 保存网站配置
  saveConfig: (id: number, data: any): any => http.Put(`/website/${id}`, data),
  // 清空日志
  clearLog: (id: number): any => http.Delete('/website/' + id + '/log'),
  // 更新备注
  updateRemark: (id: number, remark: string): any =>
    http.Post(`/website/${id}` + '/update_remark', { remark }),
  // 重置配置
  resetConfig: (id: number): any => http.Post(`/website/${id}/reset_config`),
  // 修改状态
  status: (id: number, status: boolean): any => http.Post(`/website/${id}/status`, { status }),
  // 签发证书
  obtainCert: (id: number): any => http.Post(`/website/${id}/obtain_cert`)
}
