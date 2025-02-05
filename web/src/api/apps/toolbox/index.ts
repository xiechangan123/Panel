import { request } from '@/utils'

export default {
  // DNS
  dns: (): any => request.get('/apps/toolbox/dns'),
  // 设置 DNS
  updateDns: (dns1: string, dns2: string): any => request.post('/apps/toolbox/dns', { dns1, dns2 }),
  // SWAP
  swap: (): any => request.get('/apps/toolbox/swap'),
  // 设置 SWAP
  updateSwap: (size: number): any => request.post('/apps/toolbox/swap', { size }),
  // 时区
  timezone: (): any => request.get('/apps/toolbox/timezone'),
  // 设置时区
  updateTimezone: (timezone: string): any => request.post('/apps/toolbox/timezone', { timezone }),
  // 设置时间
  updateTime: (time: string): any => request.post('/apps/toolbox/time', { time }),
  // 同步时间
  syncTime: (): any => request.post('/apps/toolbox/syncTime'),
  // 主机名
  hostname: (): any => request.get('/apps/toolbox/hostname'),
  // Hosts
  hosts: (): any => request.get('/apps/toolbox/hosts'),
  // 设置主机名
  updateHostname: (hostname: string): any => request.post('/apps/toolbox/hostname', { hostname }),
  // 设置 Hosts
  updateHosts: (hosts: string): any => request.post('/apps/toolbox/hosts', { hosts }),
  // 设置 Root 密码
  updateRootPassword: (password: string): any =>
    request.post('/apps/toolbox/rootPassword', { password })
}
