import { http } from '@/utils'

export default {
  // 面板信息
  panel: (): any => http.Get('/dashboard/panel'),
  // 首页应用
  homeApps: (): any => http.Get('/dashboard/home_apps'),
  // 实时信息
  current: (nets: string[], disks: string[]): any =>
    http.Post('/dashboard/current', { nets, disks }, { meta: { noAlert: true } }),
  // 系统信息
  systemInfo: (): any => http.Get('/dashboard/system_info'),
  // 统计信息
  countInfo: (): any => http.Get('/dashboard/count_info'),
  // 已安装的数据库和PHP
  installedDbAndPhp: (): any => http.Get('/dashboard/installed_db_and_php'),
  // 检查更新
  checkUpdate: (): any => http.Get('/dashboard/check_update'),
  // 更新日志
  updateInfo: (): any => http.Get('/dashboard/update_info'),
  // 更新面板
  update: (): any => http.Post('/dashboard/update'),
  // 重启面板
  restart: (): any => http.Post('/dashboard/restart')
}
