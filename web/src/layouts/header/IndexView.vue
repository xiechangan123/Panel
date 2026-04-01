<script lang="ts" setup>
import ReloadPage from '@/layouts/header/components/ReloadPage.vue'
import StickyNote from '@/layouts/header/components/StickyNote.vue'
import TaskStatus from '@/layouts/header/components/TaskStatus.vue'
import AppTab from '@/layouts/tab/IndexView.vue'
import FullScreen from './components/FullScreen.vue'
import ThemeMode from './components/ThemeMode.vue'
import UserAvatar from './components/UserAvatar.vue'

import MenuCollapse from '@/layouts/header/components/MenuCollapse.vue'
import { useThemeStore } from '@/stores'

const themeStore = useThemeStore()
</script>

<template>
  <div flex w-full items-center justify-between>
    <menu-collapse v-if="themeStore.isMobile" />
    <section v-if="!themeStore.isMobile && themeStore.tab.visible" pr-12 flex-1 w-0>
      <app-tab />
    </section>
    <span v-if="!themeStore.isMobile && themeStore.tab.visible" mx-6 opacity-20>|</span>
    <div ml-auto px-12 flex flex-shrink-0 items-center>
      <task-status />
      <sticky-note />
      <reload-page />
      <full-screen />
      <theme-mode />
      <user-avatar />
    </div>
  </div>
</template>
