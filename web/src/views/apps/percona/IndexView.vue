<script setup lang="ts">
defineOptions({
  name: 'apps-percona-index'
})

import copy2clipboard from '@vavt/copy2clipboard'
import { NButton, NDataTable, NInput } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import percona from '@/api/apps/percona'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')

const { data: rootPassword } = useRequest(percona.rootPassword, {
  initialData: ''
})
const { data: config } = useRequest(percona.config, {
  initialData: ''
})
const { data: slowLog } = useRequest(percona.slowLog, {
  initialData: ''
})
const { data: load } = useRequest(percona.load, {
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

const handleSaveConfig = () => {
  useRequest(percona.saveConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleClearLog = () => {
  useRequest(percona.clearLog()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleClearSlowLog = () => {
  useRequest(percona.clearSlowLog()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleSetRootPassword = async () => {
  await percona.setRootPassword(rootPassword.value)
  window.$message.success($gettext('Modified successfully'))
}

const handleCopyRootPassword = () => {
  copy2clipboard(rootPassword.value).then(() => {
    window.$message.success($gettext('Copied successfully'))
  })
}
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <n-flex vertical>
          <service-status service="mysqld" />
          <n-card :title="$gettext('Root Password')">
            <n-flex>
              <n-input-group>
                <n-input v-model:value="rootPassword" type="password" show-password-on="click" />
                <n-button type="primary" ghost @click="handleCopyRootPassword">
                  {{ $gettext('Copy') }}
                </n-button>
              </n-input-group>
              <n-button type="primary" @click="handleSetRootPassword">
                {{ $gettext('Save Changes') }}
              </n-button>
            </n-flex>
          </n-card>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Modify Configuration')">
        <n-flex vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the Percona main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!'
              )
            }}
          </n-alert>
          <common-editor v-model:value="config" height="60vh" />
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
          :columns="loadColumns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <n-button type="primary" @click="handleClearLog">
          {{ $gettext('Clear Log') }}
        </n-button>
        <realtime-log service="mysqld" />
      </n-tab-pane>
      <n-tab-pane name="slow-log" :tab="$gettext('Slow Query Log')">
        <n-button type="primary" @click="handleClearSlowLog">
          {{ $gettext('Clear Slow Log') }}
        </n-button>
        <realtime-log :path="slowLog" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
