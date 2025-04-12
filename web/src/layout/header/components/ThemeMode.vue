<script lang="ts" setup>
import { useThemeStore } from '@/store'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
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
    {{ $gettext('Switch Theme') }}
  </n-tooltip>
</template>
