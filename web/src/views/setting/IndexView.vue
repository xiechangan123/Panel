<script setup lang="ts">
defineOptions({
  name: 'setting-index'
})

import SettingBase from '@/views/setting/SettingBase.vue'
import SettingHttps from '@/views/setting/SettingHttps.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const currentTab = ref('base')
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="base" :tab="$gettext('Basic')">
        <setting-base />
      </n-tab-pane>
      <n-tab-pane name="https" tab="HTTPS">
        <setting-https />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>

<style scoped lang="scss"></style>
