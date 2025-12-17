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
        <i-mdi-weather-sunny v-if="theme.darkMode" />
        <i-mdi-weather-night v-else />
      </n-icon>
    </template>
    {{ $gettext('Switch Theme') }}
  </n-tooltip>
</template>
