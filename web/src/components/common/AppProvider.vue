<script lang="ts" setup>
import systemdlog from '@/utils/hljs/systemdlog'
import hljs from 'highlight.js/lib/core'
import log from 'highlight.js/lib/languages/accesslog'

import { useThemeStore } from '@/store'

hljs.registerLanguage('accesslog', log)
hljs.registerLanguage('systemdlog', systemdlog)

const themeStore = useThemeStore()

watch(
  () => themeStore.darkMode,
  (newValue) => {
    if (newValue) document.documentElement.classList.add('dark')
    else document.documentElement.classList.remove('dark')
  },
  {
    immediate: true
  }
)

function handleWindowResize() {
  themeStore.setIsMobile(document.body.offsetWidth <= 640)
}

onMounted(() => {
  handleWindowResize()
  window.addEventListener('resize', handleWindowResize)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', handleWindowResize)
})
</script>

<template>
  <n-config-provider
    :hljs="hljs"
    :theme="themeStore.naiveTheme"
    :locale="themeStore.naiveLocale"
    :date-locale="themeStore.naiveDateLocale"
    wh-full
  >
    <slot />
  </n-config-provider>
</template>
