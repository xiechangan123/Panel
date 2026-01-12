<script setup lang="ts">
defineOptions({
  name: 'apps-apache-index'
})

import { useGettext } from 'vue3-gettext'

import apache from '@/api/apps/apache'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')

const { data: config } = useRequest(apache.config, {
  initialData: ''
})
const { data: errorLog } = useRequest(apache.errorLog, {
  initialData: ''
})
const { data: load } = useRequest(apache.load, {
  initialData: []
})

const columns: any = [
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

const handleSaveConfig = () => {
  useRequest(apache.saveConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleClearErrorLog = () => {
  useRequest(apache.clearErrorLog()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="apache" show-reload />
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Modify Configuration')">
        <n-flex vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the %{name} main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!',
                { name: 'Apache' }
              )
            }}
          </n-alert>
          <common-editor v-model:value="config" lang="apache" height="60vh" />
          <n-flex>
            <n-button type="primary" @click="handleSaveConfig">
              {{ $gettext('Save') }}
            </n-button>
          </n-flex>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="load" :tab="$gettext('Load Status')">
        <n-data-table
          striped
          remote
          :scroll-x="400"
          :loading="false"
          :columns="columns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="apache" />
      </n-tab-pane>
      <n-tab-pane name="error-log" :tab="$gettext('Error Logs')">
        <n-flex vertical>
          <n-flex>
            <n-button type="primary" @click="handleClearErrorLog">
              {{ $gettext('Clear Log') }}
            </n-button>
          </n-flex>
          <realtime-log :path="errorLog" />
        </n-flex>
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
