import '@/styles/index.scss'
import '@/styles/reset.css'
import '@vue-js-cron/naive-ui/dist/naive-ui.css'
import 'virtual:uno.css'

import { createApp } from 'vue'
import App from './App.vue'

import { setupRouter } from '@/router'
import { setupStore, useThemeStore } from '@/store'
import { gettext, setCurrent, setupNaiveDiscreteApi } from '@/utils'

import home from '@/api/panel/home'
import CronNaivePlugin from '@vue-js-cron/naive-ui'

async function loadMonacoLocale(locale: string) {
  try {
    switch (locale) {
      case 'zh_CN':
        await import('monaco-editor/esm/nls.messages.zh-cn.js')
        break
      case 'zh_TW':
        await import('monaco-editor/esm/nls.messages.zh-tw.js')
        break
      default:
        // 英语不需要加载
        break
    }
  } catch (error) {
    console.warn(`Failed to load monaco-editor locale: ${locale}`, error)
  }
}

async function setupMonacoEditor(app: any) {
  const [editorWorker, jsonWorker, cssWorker, htmlWorker, tsWorker] = await Promise.all([
    import('monaco-editor/esm/vs/editor/editor.worker?worker'),
    import('monaco-editor/esm/vs/language/json/json.worker?worker'),
    import('monaco-editor/esm/vs/language/css/css.worker?worker'),
    import('monaco-editor/esm/vs/language/html/html.worker?worker'),
    import('monaco-editor/esm/vs/language/typescript/ts.worker?worker')
  ])

  self.MonacoEnvironment = {
    getWorker(_: any, label: string) {
      if (label === 'json') {
        return new jsonWorker.default()
      }
      if (label === 'css' || label === 'scss' || label === 'less') {
        return new cssWorker.default()
      }
      if (label === 'html' || label === 'handlebars' || label === 'razor') {
        return new htmlWorker.default()
      }
      if (label === 'typescript' || label === 'javascript') {
        return new tsWorker.default()
      }
      return new editorWorker.default()
    }
  }

  const [{ install: VueMonacoEditorPlugin }, monaco] = await Promise.all([
    import('@guolao/vue-monaco-editor'),
    import('monaco-editor')
  ])

  app.use(VueMonacoEditorPlugin, {
    monaco
  })
}

async function setupApp() {
  const app = createApp(App)
  app.use(CronNaivePlugin)
  await setupStore(app)
  await setupNaiveDiscreteApi()

  await setupPanel().then(() => {
    app.use(gettext)
  })

  await setupMonacoEditor(app)

  await setupRouter(app)
  app.mount('#app')
}

const setupPanel = async () => {
  const themeStore = useThemeStore()
  setCurrent(themeStore.locale)

  return new Promise<void>((resolve) => {
    useRequest(home.panel, {
      initialData: {
        name: import.meta.env.VITE_APP_TITLE,
        locale: 'en'
      }
    }).onSuccess(async ({ data }: { data: any }) => {
      await loadMonacoLocale(data.locale)

      setCurrent(data.locale)
      themeStore.setLocale(data.locale)
      themeStore.setName(data.name)

      resolve()
    })
  })
}

setupApp()
