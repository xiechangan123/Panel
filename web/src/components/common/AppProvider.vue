<script lang="ts" setup>
import { useThemeStore } from '@/store'

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
    :theme="themeStore.naiveTheme"
    :locale="themeStore.naiveLocale"
    :date-locale="themeStore.naiveDateLocale"
    wh-full
  >
    <slot />
  </n-config-provider>
</template>
