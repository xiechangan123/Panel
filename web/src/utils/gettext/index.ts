import type { App } from 'vue'
import { createGettext } from 'vue3-gettext'

let gettext: ReturnType<typeof createGettext>

export function setupGettext(app: App) {
  gettext = createGettext({
    availableLanguages: {
      en: 'English',
      zh_CN: '简体中文',
      zh_TW: '繁體中文'
    },
    defaultLanguage: 'zh_CN'
  })
  app.use(gettext)
}
