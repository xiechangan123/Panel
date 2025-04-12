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
    defaultLanguage: 'zh_CN',
    globalProperties: {
      gettext: ['$gettext', '__'], // 这样支持同时使用 $gettext, __ 两种方式
      ngettext: ['$ngettext', '_n'],
      pgettext: ['$pgettext', '_x'],
      npgettext: ['$npgettext', '_nx']
    }
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
