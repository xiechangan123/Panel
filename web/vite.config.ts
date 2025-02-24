import { defineConfig, loadEnv } from 'vite'

import { createViteProxy, viteDefine } from './build/config'
import { setupVitePlugins } from './build/plugins'
import { convertEnv, getRootPath, getSrcPath } from './build/utils'

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
      chunkSizeWarningLimit: 1024, // chunk 大小警告的限制（单位kb）
      commonjsOptions: {
        ignoreTryCatch: false
      }
    }
  }
})
