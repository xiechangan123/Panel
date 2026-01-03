<script lang="ts" setup>
defineOptions({
  name: 'app-index'
})

import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import app from '@/api/panel/app'
import AllView from '@/views/app/AllView.vue'

const { $gettext } = useGettext()

const currentTab = ref('all')

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
          <n-tab name="all" :tab="$gettext('All')" />
          <n-tab name="environment" :tab="$gettext('Environment')" />
          <n-tab name="compose" :tab="$gettext('Compose Templates')" />
        </n-tabs>
        <n-button v-if="currentTab != 'installed'" type="primary" @click="handleUpdateCache">
          {{ $gettext('Update Cache') }}
        </n-button>
      </div>
    </template>
    <all-view v-if="currentTab === 'all'" />
  </common-page>
</template>
