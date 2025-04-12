<script setup lang="ts">
import { useThemeStore } from '@/store'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const themeStore = useThemeStore()
</script>

<template>
  <n-tooltip trigger="hover">
    <template #trigger>
      <n-color-picker
        class="mr-16 h-24 w-24"
        :show-alpha="false"
        v-model:value="themeStore.primaryColor"
        :render-label="() => ''"
      />
    </template>
    {{ $gettext('Set Theme Color') }}
  </n-tooltip>
</template>
