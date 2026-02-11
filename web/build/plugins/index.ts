import type { PluginOption } from 'vite'
import vue from '@vitejs/plugin-vue'
import unocss from 'unocss/vite'
import { compression } from 'vite-plugin-compression2'
import vueDevTools from 'vite-plugin-vue-devtools'

import { setupHtmlPlugin } from './html'
import unplugins from './unplugin'

export function setupVitePlugins(viteEnv: ViteEnv): PluginOption[] {
  return [
    vue(),
    vueDevTools(),
    ...unplugins,
    unocss(),
    setupHtmlPlugin(viteEnv),
    compression({
      algorithms: ['brotliCompress'],
      deleteOriginalAssets: true,
      skipIfLargerOrEqual: true
    })
  ]
}
