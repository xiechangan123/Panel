import { http } from '@/utils'

export default {
  // 负载状态
  load: (): any => http.Get('/apps/mysql/load'),
  // 获取配置
  config: (): any => http.Get('/apps/mysql/config'),
  // 保存配置
  saveConfig: (config: string): any => http.Post('/apps/mysql/config', { config }),
  // 获取慢查询日志
  slowLog: (): any => http.Get('/apps/mysql/slow_log'),
  // 获取 root 密码
  rootPassword: (): any => http.Get('/apps/mysql/root_password'),
  // 修改 root 密码
  setRootPassword: (password: string): any => http.Post('/apps/mysql/root_password', { password }),
  // 获取配置调整参数
  configTune: (): any => http.Get('/apps/mysql/config_tune'),
  // 保存配置调整参数
  saveConfigTune: (data: any): any => http.Post('/apps/mysql/config_tune', data),
  // 性能
  processes: (): any => http.Get('/apps/mysql/processes'),
  killProcess: (id: number): any => http.Post(`/apps/mysql/processes/${id}/kill`),
  transactions: (): any => http.Get('/apps/mysql/transactions'),
  topSQL: (): any => http.Get('/apps/mysql/top_sql'),
  enableTopSQL: (): any => http.Post('/apps/mysql/top_sql/enable'),
  resetTopSQL: (): any => http.Post('/apps/mysql/top_sql/reset'),
  // 维护
  tables: (): any => http.Get('/apps/mysql/tables'),
  runMaintenance: (data: any): any => http.Post('/apps/mysql/maintenance', data),
  binlogs: (): any => http.Get('/apps/mysql/binlogs'),
  purgeBinlog: (file: string): any => http.Post('/apps/mysql/binlogs/purge', { file }),
  replication: (): any => http.Get('/apps/mysql/replication'),
}
