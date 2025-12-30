import '@/styles/index.scss'
import '@/styles/reset.css'
import 'virtual:uno.css'

import { createApp } from 'vue'
import App from './App.vue'

import { setupRouter } from '@/router'
import { setupStore, useThemeStore } from '@/store'
import { gettext, setCurrent, setupNaiveDiscreteApi } from '@/utils'

import home from '@/api/panel/home'

async function setupApp() {
  const app = createApp(App)
  await setupStore(app)
  await setupNaiveDiscreteApi()

  await setupPanel().then(() => {
    app.use(gettext)
  })

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
      setCurrent(data.locale)
      themeStore.setLocale(data.locale)
      themeStore.setName(data.name)

      resolve()
    })
  })
}

setupApp()
