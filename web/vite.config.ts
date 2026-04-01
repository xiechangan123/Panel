import { defineConfig, loadEnv } from 'vite'

import { viteDefine } from './config/define'
import { setupVitePlugins } from './config/plugins'
import { createViteProxy } from './config/proxy'
import { convertEnv, getRootPath, getSrcPath } from './config/utils'

export default defineConfig(({ mode }) => {
  const srcPath = getSrcPath()
  const rootPath = getRootPath()

  const viteEnv = convertEnv(loadEnv(mode, process.cwd()))

  const { VITE_PORT, VITE_PUBLIC_PATH, VITE_USE_PROXY, VITE_PROXY_TYPE } = viteEnv
  return {
    base: VITE_PUBLIC_PATH,
    resolve: {
      alias: {
        '~': rootPath,
        '@': srcPath
      }
    },
    define: viteDefine,
    plugins: setupVitePlugins(viteEnv),
    server: {
      host: '0.0.0.0',
      port: VITE_PORT,
      open: false,
      proxy: createViteProxy(VITE_USE_PROXY, VITE_PROXY_TYPE as ProxyType)
    },
    build: {
      reportCompressedSize: false,
      sourcemap: false,
      chunkSizeWarningLimit: 1024
    }
  }
})
