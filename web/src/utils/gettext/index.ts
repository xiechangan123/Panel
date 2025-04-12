import { createGettext as vue3Gettext } from 'vue3-gettext'

import translations from '@/locales/translations.json'

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

export function createGettext(): any {
  gettext = vue3Gettext({
    availableLanguages: {
      en: 'English',
      zh_CN: '简体中文',
      zh_TW: '繁體中文'
    },
    defaultLanguage: 'zh_CN',
    translations: translations
  })

  return gettext
}
