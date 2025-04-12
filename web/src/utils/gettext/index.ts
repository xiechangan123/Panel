import type { App } from 'vue'
import { createGettext as vue3Gettext } from 'vue3-gettext'

export let gettext: ReturnType<typeof vue3Gettext>

export function $gettext(msgid: string, params?: Record<string, string | number>) {
  return gettext.$gettext(msgid, params)
}

export function $ngettext(
  msgid: string,
  plural: string,
  n: number,
  params?: Record<string, string | number>
) {
  return gettext.$ngettext(msgid, plural, n, params)
}

export function setupGettext(app: App) {
  gettext = vue3Gettext({
    availableLanguages: {
      en: 'English',
      zh_CN: '简体中文',
      zh_TW: '繁體中文'
    },
    defaultLanguage: 'zh_CN'
  })
  app.use(gettext)
}

export function createGettext(): any {
  gettext = vue3Gettext({
    availableLanguages: {
      en: 'English',
      zh_CN: '简体中文',
      zh_TW: '繁體中文'
    },
    defaultLanguage: 'zh_CN'
  })

  return gettext
}
