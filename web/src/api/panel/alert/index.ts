import { http } from '@/utils'

export default {
  // 告警规则列表
  rules: (page: number, limit: number): any => http.Get('/alert/rule', { params: { page, limit } }),
  // 新增告警规则
  createRule: (data: any): any => http.Post('/alert/rule', data),
  // 更新告警规则
  updateRule: (id: number, data: any): any => http.Put(`/alert/rule/${id}`, data),
  // 删除告警规则
  deleteRule: (id: number): any => http.Delete(`/alert/rule/${id}`),
  // 告警记录列表
  records: (page: number, limit: number): any =>
    http.Get('/alert/record', { params: { page, limit } }),
  // 清空告警记录
  clearRecords: (): any => http.Post('/alert/record/clear'),
}
