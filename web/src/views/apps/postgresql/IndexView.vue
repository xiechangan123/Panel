<script setup lang="ts">
defineOptions({
  name: 'apps-postgresql-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import postgresql from '@/api/apps/postgresql'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')

const { data: log } = useRequest(postgresql.log, {
  initialData: ''
})
const { data: config } = useRequest(postgresql.config, {
  initialData: ''
})
const { data: userConfig } = useRequest(postgresql.userConfig, {
  initialData: ''
})
const { data: load } = useRequest(postgresql.load, {
  initialData: []
})

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

const handleSaveConfig = async () => {
  await postgresql.saveConfig(config.value)
  window.$message.success($gettext('Saved successfully'))
}

const handleSaveUserConfig = async () => {
  await postgresql.saveUserConfig(userConfig.value)
  window.$message.success($gettext('Saved successfully'))
}

const handleClearLog = async () => {
  await postgresql.clearLog()
  window.$message.success($gettext('Cleared successfully'))
}
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="postgresql" show-reload />
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Main Configuration')">
        <n-flex vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the PostgreSQL main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!'
              )
            }}
          </n-alert>
          <Editor
            v-model:value="config"
            language="ini"
            theme="vs-dark"
            height="60vh"
            mt-8
            :options="{
              automaticLayout: true,
              smoothScrolling: true
            }"
          />
          <n-flex>
            <n-button type="primary" @click="handleSaveConfig">
              {{ $gettext('Save') }}
            </n-button>
          </n-flex>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="user-config" :tab="$gettext('User Configuration')">
        <n-flex vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the PostgreSQL user configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!'
              )
            }}
          </n-alert>
          <Editor
            v-model:value="userConfig"
            language="ini"
            theme="vs-dark"
            height="60vh"
            mt-8
            :options="{
              automaticLayout: true,
              smoothScrolling: true
            }"
          />
          <n-flex>
            <n-button type="primary" @click="handleSaveUserConfig">
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
          :columns="loadColumns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <n-flex vertical>
          <n-flex>
            <n-button type="primary" @click="handleClearLog">
              {{ $gettext('Clear Log') }}
            </n-button>
          </n-flex>
          <realtime-log service="postgresql" />
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="slow-log" :tab="$gettext('Slow Logs')">
        <realtime-log :path="log" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
