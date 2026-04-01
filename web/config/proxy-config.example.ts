const proxyConfigMappings: Record<ProxyType, ProxyConfig[]> = {
  dev: [
    {
      prefix: '/api/ws',
      target: 'ws://localhost:8888/api/ws',
      changeOrigin: true,
      secure: false,
      ws: true
    },
    {
      prefix: '/api',
      target: 'http://localhost:8080/api',
      changeOrigin: true,
      secure: false
    }
  ],
  test: [
    {
      prefix: '/api',
      target: 'http://localhost:8080/api'
    }
  ],
  prod: [
    {
      prefix: '/api',
      target: 'http://localhost:8080/api'
    }
  ]
}

export function getProxyConfigs(envType: ProxyType = 'dev'): ProxyConfig[] {
  return proxyConfigMappings[envType]
}
