<script setup lang="ts">
defineOptions({
  name: 'apps-memcached-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import memcached from '@/api/apps/memcached'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')

const loadColumns: any = [
  {
    title: $gettext('Property'),
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Current Value'),
    key: 'value',
    minWidth: 200,
    ellipsis: { tooltip: true }
  }
]

const { data: load } = useRequest(memcached.load, {
  initialData: []
})

const { data: config } = useRequest(memcached.config, {
  initialData: {
    config: ''
  }
})

const handleSaveConfig = () => {
  useRequest(memcached.updateConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button
        v-if="currentTab == 'config'"
        class="ml-16"
        type="primary"
        @click="handleSaveConfig"
      >
        <the-icon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="memcached" />
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Service Configuration')">
        <n-space vertical>
          <Editor
            v-model:value="config"
            language="ini"
            theme="vs-dark"
            height="60vh"
            mt-8
            :options="{
              automaticLayout: true,
              formatOnType: true,
              formatOnPaste: true
            }"
          />
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="load" :tab="$gettext('Load Status')">
        <n-data-table
          striped
          remote
          :scroll-x="1000"
          :loading="false"
          :columns="loadColumns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="memcached" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
