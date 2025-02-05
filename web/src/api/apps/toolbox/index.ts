import { http } from '@/utils'

export default {
  // DNS
  dns: (): any => http.Get('/apps/toolbox/dns'),
  // 设置 DNS
  updateDns: (dns1: string, dns2: string): any => http.Post('/apps/toolbox/dns', { dns1, dns2 }),
  // SWAP
  swap: (): any => http.Get('/apps/toolbox/swap'),
  // 设置 SWAP
  updateSwap: (size: number): any => http.Post('/apps/toolbox/swap', { size }),
  // 时区
  timezone: (): any => http.Get('/apps/toolbox/timezone'),
  // 设置时区
  updateTimezone: (timezone: string): any => http.Post('/apps/toolbox/timezone', { timezone }),
  // 设置时间
  updateTime: (time: string): any => http.Post('/apps/toolbox/time', { time }),
  // 同步时间
  syncTime: (): any => http.Post('/apps/toolbox/syncTime'),
  // 主机名
  hostname: (): any => http.Get('/apps/toolbox/hostname'),
  // Hosts
  hosts: (): any => http.Get('/apps/toolbox/hosts'),
  // 设置主机名
  updateHostname: (hostname: string): any => http.Post('/apps/toolbox/hostname', { hostname }),
  // 设置 Hosts
  updateHosts: (hosts: string): any => http.Post('/apps/toolbox/hosts', { hosts }),
  // 设置 Root 密码
  updateRootPassword: (password: string): any =>
    http.Post('/apps/toolbox/rootPassword', { password })
}
