<script lang="ts" setup>
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const { isFullscreen, toggle } = useFullscreen()
</script>

<template>
  <n-tooltip trigger="hover">
    <template #trigger>
      <n-icon mr-20 cursor-pointer size="20" @click="toggle">
        <icon-ant-design:fullscreen-exit-outlined v-if="isFullscreen" />
        <icon-ant-design:fullscreen-outlined v-else />
      </n-icon>
    </template>
    {{ $gettext('Fullscreen Display') }}
  </n-tooltip>
</template>
