import { http, request } from '@/utils'

import type { RequestConfig } from '~/types/axios'

export default {
  // 面板信息
  panel: (): Promise<Response> => fetch('/api/dashboard/panel'),
  // 面板菜单
  menu: (): any => request.get('/dashboard/menu'),
  // 首页应用
  homeApps: (): any => request.get('/dashboard/homeApps'),
  // 实时信息
  current: (nets: string[], disks: string[]): any =>
    request.post('/dashboard/current', { nets, disks }, { noNeedTip: true } as RequestConfig),
  // 系统信息
  systemInfo: (): any => request.get('/dashboard/systemInfo'),
  // 统计信息
  countInfo: (): any => request.get('/dashboard/countInfo'),
  // 已安装的数据库和PHP
  installedDbAndPhp: (): any => request.get('/dashboard/installedDbAndPhp'),
  // 检查更新
  checkUpdate: (): any => request.get('/dashboard/checkUpdate'),
  // 更新日志
  updateInfo: (): any => request.get('/dashboard/updateInfo'),
  // 更新面板
  update: (): any => request.post('/dashboard/update', null),
  // 重启面板
  restart: (): any => request.post('/dashboard/restart')
}

export const panel = () => http.Get('/dashboard/panel')
export const current = (nets: string[], disks: string[]) =>
  http.Post('/dashboard/current', { nets, disks }, { meta: { noAlert: true } })
