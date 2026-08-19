import { http } from '@/utils'

export default {
  // 负载状态
  load: (): any => http.Get('/apps/mariadb/load'),
  // 获取配置
  config: (): any => http.Get('/apps/mariadb/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/mariadb/config', { config }),
  // 获取慢查询日志
  slowLog: (): any => http.Get('/apps/mariadb/slow_log'),
  // 获取 root 密码
  rootPassword: (): any => http.Get('/apps/mariadb/root_password'),
  // 修改 root 密码
  setRootPassword: (password: string): any =>
    http.Post('/apps/mariadb/root_password', { password }),
  // 获取配置调整参数
  configTune: (): any => http.Get('/apps/mariadb/config_tune'),
  // 保存配置调整参数
  saveConfigTune: (data: any): any => http.Post('/apps/mariadb/config_tune', data),
  // 性能
  processes: (): any => http.Get('/apps/mariadb/processes'),
  killProcess: (id: number): any => http.Post(`/apps/mariadb/processes/${id}/kill`),
  transactions: (): any => http.Get('/apps/mariadb/transactions'),
  topSQL: (): any => http.Get('/apps/mariadb/top_sql'),
  enableTopSQL: (): any => http.Post('/apps/mariadb/top_sql/enable'),
  resetTopSQL: (): any => http.Post('/apps/mariadb/top_sql/reset'),
  // 维护
  tables: (): any => http.Get('/apps/mariadb/tables'),
  runMaintenance: (data: any): any => http.Post('/apps/mariadb/maintenance', data),
  binlogs: (): any => http.Get('/apps/mariadb/binlogs'),
  purgeBinlog: (file: string): any => http.Post('/apps/mariadb/binlogs/purge', { file }),
  replication: (): any => http.Get('/apps/mariadb/replication'),
}
