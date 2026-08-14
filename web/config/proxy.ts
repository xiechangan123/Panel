import type { ProxyOptions } from 'vite'
import { getProxyConfigs } from './proxy-config.ts'

export function createViteProxy(isUseProxy = true, proxyType: ProxyType) {
  if (!isUseProxy) return undefined

  const proxyConfigs = getProxyConfigs(proxyType)
  const proxy: Record<string, string | ProxyOptions> = {}

  proxyConfigs.forEach((proxyConfig) => {
    proxy[proxyConfig.prefix] = {
      target: proxyConfig.target,
      secure: proxyConfig.secure,
      changeOrigin: true,
      rewrite: (path: string) => path.replace(new RegExp(`^${proxyConfig.prefix}`), '')
    }
  })

  return proxy
}
