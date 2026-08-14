import { http } from '@/utils'

export interface NetworkListParams {
  page: number
  limit: number
  sort?: string
  order?: string
  state?: string // 逗号分隔
  pid?: string
  process?: string
  port?: string
}

export interface NetworkFamilyConfig {
  mode: 'auto' | 'manual' | 'disabled'
  addresses: string[]
  gateway: string
  auto_dns: boolean
  dns: string[]
}

export interface NetworkInterfaceConfig {
  name: string
  mtu: number
  ipv4: NetworkFamilyConfig
  ipv6: NetworkFamilyConfig
}

export interface NetworkInterface extends Omit<NetworkInterfaceConfig, 'mtu'> {
  type: string
  state: string
  mac: string
  current_mtu: number
  configured_mtu: number
  current_ipv4: string[]
  current_ipv6: string[]
  editable: boolean
  reason: string
}

export interface NetworkInterfaces {
  manager: string
  items: NetworkInterface[]
  pending: boolean // 是否有变更等待确认
}

export default {
  list: (params: NetworkListParams) => http.Get('/toolbox_network/list', { params }),
  interfaces: (): any => http.Get('/toolbox_network/interfaces'),
  updateInterface: (config: NetworkInterfaceConfig): any =>
    http.Post('/toolbox_network/interfaces', config),
  confirmInterface: (): any => http.Post('/toolbox_network/interfaces/confirm'),
  rollbackInterface: (): any => http.Post('/toolbox_network/interfaces/rollback')
}
