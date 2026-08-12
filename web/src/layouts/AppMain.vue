<script lang="ts" setup>
import { useTabStore } from '@/stores'

const tabStore = useTabStore()
</script>

<template>
  <div class="flex flex-col wh-full">
    <router-view v-slot="{ Component, route }">
      <keep-alive
        v-for="tab in tabStore.tabs"
        :key="tab.path"
        :include="tab.keepAlive ? undefined : []"
      >
        <component
          :is="Component"
          v-if="route.path === tab.path && !tabStore.reloading"
          :key="route.path"
        />
      </keep-alive>
    </router-view>
  </div>
</template>
