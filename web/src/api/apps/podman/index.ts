import { request } from '@/utils'

export default {
  // 获取注册表配置
  registryConfig: (): any => request.get('/apps/podman/registryConfig'),
  // 保存注册表配置
  saveRegistryConfig: (config: string): any =>
    request.post('/apps/podman/registryConfig', { config }),
  // 获取存储配置
  storageConfig: (): any => request.get('/apps/podman/storageConfig'),
  // 保存存储配置
  saveStorageConfig: (config: string): any => request.post('/apps/podman/storageConfig', { config })
}
