<script lang="ts" setup>
defineOptions({
  name: 'app-index'
})

import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import app from '@/api/panel/app'
import InstallView from '@/views/app/InstallView.vue'

const { $gettext } = useGettext()

const currentTab = ref('installed')

const handleUpdateCache = () => {
  useRequest(app.updateCache()).onSuccess(() => {
    window.$message.success($gettext('Cache updated successfully'))
  })
}
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <div class="flex items-center justify-between">
        <n-tabs v-model:value="currentTab" animated class="flex-1">
          <n-tab name="installed" :tab="$gettext('Installed')" />
          <n-tab name="install" :tab="$gettext('Install')" />
          <n-tab name="environment" :tab="$gettext('Environment')" />
          <n-tab name="compose" :tab="$gettext('Compose Templates')" />
        </n-tabs>
        <n-button v-if="currentTab != 'installed'" type="primary" @click="handleUpdateCache">
          {{ $gettext('Update Cache') }}
        </n-button>
      </div>
    </template>
    <install-view v-if="currentTab === 'install'" />
  </common-page>
</template>
