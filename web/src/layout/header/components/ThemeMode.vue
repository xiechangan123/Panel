<script lang="ts" setup>
import { useThemeStore } from '@/store'

const theme = useThemeStore()
</script>

<template>
  <n-tooltip trigger="hover">
    <template #trigger>
      <n-icon mr-20 cursor-pointer size="20" @click="theme.toggleDarkMode">
        <icon-mdi-moon-waning-crescent v-if="theme.darkMode" />
        <icon-mdi-white-balance-sunny v-else />
      </n-icon>
    </template>
    切换主题
  </n-tooltip>
</template>
