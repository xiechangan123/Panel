<script lang="ts" setup>
defineOptions({
  name: 'website-index'
})

import PhpView from '@/views/website/PhpView.vue'
import SettingView from '@/views/website/SettingView.vue'

const currentTab = ref('proxy')
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="proxy" :tab="$gettext('Reverse Proxy')" />
        <n-tab name="php" :tab="$gettext('PHP')" />
        <n-tab name="static" :tab="$gettext('Pure Static')" />
        <n-tab name="setting" :tab="$gettext('Settings')" />
      </n-tabs>
    </template>
    <php-view v-if="currentTab === 'php'" />
    <setting-view v-if="currentTab === 'setting'" />
  </common-page>
</template>
