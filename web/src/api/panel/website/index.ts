import type { AxiosResponse } from 'axios'

import { request } from '@/utils'

export default {
  // 列表
  list: (page: number, limit: number): Promise<AxiosResponse<any>> =>
    request.get('/website', { params: { page, limit } }),
  // 添加
  add: (data: any): Promise<AxiosResponse<any>> => request.post('/website', data),
  // 删除
  delete: (data: any): Promise<AxiosResponse<any>> => request.post('/website/' + data.id, data),
  // 获取默认配置
  defaultConfig: (): Promise<AxiosResponse<any>> => request.get('/website/defaultConfig'),
  // 保存默认配置
  saveDefaultConfig: (index: string, stop: string): Promise<AxiosResponse<any>> =>
    request.post('/website/defaultConfig', { index, stop }),
  // 网站配置
  config: (id: number): Promise<AxiosResponse<any>> => request.get('/websites/' + id + '/config'),
  // 保存网站配置
  saveConfig: (id: number, data: any): Promise<AxiosResponse<any>> =>
    request.post('/websites/' + id + '/config', data),
  // 清空日志
  clearLog: (id: number): Promise<AxiosResponse<any>> => request.delete('/websites/' + id + '/log'),
  // 更新备注
  updateRemark: (id: number, remark: string): Promise<AxiosResponse<any>> =>
    request.post('/websites/' + id + '/updateRemark', { remark }),
  // 获取备份列表
  backupList: (page: number, limit: number): Promise<AxiosResponse<any>> =>
    request.get('/website/backupList', { params: { page, limit } }),
  // 创建备份
  createBackup: (id: number): Promise<AxiosResponse<any>> =>
    request.post('/websites/' + id + '/createBackup', {}),
  // 上传备份
  uploadBackup: (data: any): Promise<AxiosResponse<any>> =>
    request.put('/website/uploadBackup', data),
  // 删除备份
  deleteBackup: (name: string): Promise<AxiosResponse<any>> =>
    request.delete('/website/deleteBackup', { data: { name } }),
  // 恢复备份
  restoreBackup: (id: number, name: number): Promise<AxiosResponse<any>> =>
    request.post('/websites/' + id + '/restoreBackup', { name }),
  // 重置配置
  resetConfig: (id: number): Promise<AxiosResponse<any>> =>
    request.post('/websites/' + id + '/resetConfig'),
  // 修改状态
  status: (id: number, status: boolean): Promise<AxiosResponse<any>> =>
    request.post('/websites/' + id + '/status', { status })
}
