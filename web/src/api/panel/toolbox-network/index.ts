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

// NetworkInterfaceState 网卡当前生效的状态，用于自动获取时回填表单
export interface NetworkInterfaceState {
  addresses: string[]
  gateway: string
  dns: string[]
}

export interface NetworkInterface extends Omit<NetworkInterfaceConfig, 'mtu'> {
  type: string
  state: string
  mac: string
  current_mtu: number
  configured_mtu: number
  current_ipv4: NetworkInterfaceState
  current_ipv6: NetworkInterfaceState
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
