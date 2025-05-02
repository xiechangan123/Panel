import { http } from '@/utils'

export default {
  // 获取防火墙状态
  status: (): any => http.Get('/firewall/status'),
  // 设置防火墙状态
  updateStatus: (status: boolean): any => http.Post('/firewall/status', { status }),
  // 获取防火墙规则
  rules: (page: number, limit: number): any =>
    http.Get('/firewall/rule', { params: { page, limit } }),
  // 创建防火墙规则
  createRule: (rule: any): any => http.Post('/firewall/rule', rule),
  // 删除防火墙规则
  deleteRule: (rule: any): any => http.Delete('/firewall/rule', rule),
  // 获取防火墙IP规则
  ipRules: (page: number, limit: number): any =>
    http.Get('/firewall/ip_rule', { params: { page, limit } }),
  // 创建防火墙IP规则
  createIpRule: (rule: any): any => http.Post('/firewall/ip_rule', rule),
  // 删除防火墙IP规则
  deleteIpRule: (rule: any): any => http.Delete('/firewall/ip_rule', rule),
  // 获取防火墙转发规则
  forwards: (page: number, limit: number): any =>
    http.Get('/firewall/forward', { params: { page, limit } }),
  // 创建防火墙转发规则
  createForward: (rule: any): any => http.Post('/firewall/forward', rule),
  // 删除防火墙转发规则
  deleteForward: (rule: any): any => http.Delete('/firewall/forward', rule)
}
