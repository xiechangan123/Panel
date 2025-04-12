import { createGettext as vue3Gettext } from 'vue3-gettext'

import translations from '@/locales/translations.json'

export const locales = {
  en: 'English',
  zh_CN: '简体中文',
  zh_TW: '繁體中文'
}

export const gettext: any = vue3Gettext({
  availableLanguages: locales,
  defaultLanguage: 'zh_CN',
  translations: translations,
  silent: true
})

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

export function setCurrent(language: string) {
  gettext.current = language
}

// 变通方法，由于 gettext 不能直接对动态标题进行翻译
export function translateTitle(key: string): string {
  const titles: { [key: string]: string } = {
    // 主菜单标题
    Apps: $gettext('Apps'),
    Backup: $gettext('Backup'),
    Certificate: $gettext('Certificate'),
    Container: $gettext('Container'),
    Dashboard: $gettext('Dashboard'),
    Update: $gettext('Update'),
    Database: $gettext('Database'),
    Files: $gettext('Files'),
    Firewall: $gettext('Firewall'),
    Monitoring: $gettext('Monitoring'),
    Settings: $gettext('Settings'),
    Terminal: $gettext('Terminal'),
    Tasks: $gettext('Tasks'),
    Website: $gettext('Website'),
    // 应用标题
    'Rat Benchmark': $gettext('Rat Benchmark'),
    Toolbox: $gettext('Toolbox')
  }

  return titles[key] || key
}
