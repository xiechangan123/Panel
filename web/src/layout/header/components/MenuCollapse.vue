<script lang="ts" setup>
import { useThemeStore } from '@/store'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const themeStore = useThemeStore()
</script>

<template>
  <n-tooltip trigger="hover">
    <template #trigger>
      <n-icon cursor-pointer size="22" @click="themeStore.toggleCollapsed()">
        <icon-mdi:format-indent-increase v-if="themeStore.sider.collapsed" />
        <icon-mdi:format-indent-decrease v-else />
      </n-icon>
    </template>
    {{ $gettext('Menu Zoom') }}
  </n-tooltip>
</template>
