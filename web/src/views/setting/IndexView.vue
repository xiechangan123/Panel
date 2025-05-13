<script setup lang="ts">
import setting from '@/api/panel/setting'

defineOptions({
  name: 'setting-index'
})

import TheIcon from '@/components/custom/TheIcon.vue'
import { useThemeStore } from '@/store'
import SettingBase from '@/views/setting/SettingBase.vue'
import SettingSafe from '@/views/setting/SettingSafe.vue'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const themeStore = useThemeStore()
const currentTab = ref('base')

const { data: model } = useRequest(setting.list, {
  initialData: {
    name: '',
    channel: 'stable',
    locale: 'en',
    channel: 'stable',
    username: '',
    password: '',
    email: '',
    port: 8888,
    entrance: '',
    offline_mode: false,
    two_fa: false,
    lifetime: 0,
    bind_domain: [],
    bind_ip: [],
    bind_ua: [],
    website_path: '',
    backup_path: '',
    https: false,
    cert: '',
    key: ''
  }
})

const handleSave = () => {
  useRequest(setting.update(model.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
    if (model.value.locale !== themeStore.locale) {
      themeStore.setLocale(model.value.locale)
      window.$message.info($gettext('Panel is restarting, page will refresh in 3 seconds'))
      setTimeout(() => {
        window.location.reload()
      }, 3000)
    }
  })
}
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button type="primary" @click="handleSave">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="base" :tab="$gettext('Basic')">
        <setting-base v-model:model="model" />
      </n-tab-pane>
      <n-tab-pane name="safe" :tab="$gettext('Safe')">
        <setting-safe v-model:model="model" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>

<style scoped lang="scss"></style>
