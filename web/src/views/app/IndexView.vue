<script lang="ts" setup>
defineOptions({
  name: 'app-index'
})

import InstallView from '@/views/app/InstallView.vue'

const currentTab = ref('installed')
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="installed" :tab="$gettext('Installed')" />
        <n-tab name="install" :tab="$gettext('Install')" />
        <n-tab name="environment" :tab="$gettext('Environment')" />
        <n-tab name="compose" :tab="$gettext('Compose Templates')" />
      </n-tabs>
    </template>
    <install-view v-if="currentTab === 'install'" />
  </common-page>
</template>
