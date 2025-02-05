import { http } from '@/utils'

export default {
  // 获取注册表配置
  registryConfig: (): any => http.Get('/apps/podman/registryConfig'),
  // 保存注册表配置
  saveRegistryConfig: (config: string): any => http.Post('/apps/podman/registryConfig', { config }),
  // 获取存储配置
  storageConfig: (): any => http.Get('/apps/podman/storageConfig'),
  // 保存存储配置
  saveStorageConfig: (config: string): any => http.Post('/apps/podman/storageConfig', { config })
}
