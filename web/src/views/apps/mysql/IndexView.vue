<script setup lang="ts">
defineOptions({
  name: 'apps-mysql-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import mysql from '@/api/apps/mysql'
import systemctl from '@/api/panel/systemctl'

const { $gettext } = useGettext()
const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)

const { data: rootPassword } = useRequest(mysql.rootPassword, {
  initialData: ''
})
const { data: config } = useRequest(mysql.config, {
  initialData: ''
})
const { data: slowLog } = useRequest(mysql.slowLog, {
  initialData: ''
})
const { data: load } = useRequest(mysql.load, {
  initialData: []
})

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? $gettext('Running') : $gettext('Stopped')
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

const getStatus = async () => {
  status.value = await systemctl.status('mysqld')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('mysqld')
}

const handleSaveConfig = () => {
  useRequest(mysql.saveConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleClearErrorLog = () => {
  useRequest(mysql.clearErrorLog()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleClearSlowLog = () => {
  useRequest(mysql.clearSlowLog()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('mysqld')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('mysqld')
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
}

const handleStart = async () => {
  await systemctl.start('mysqld')
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop('mysqld')
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('mysqld')
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleSetRootPassword = async () => {
  await mysql.setRootPassword(rootPassword.value)
  window.$message.success($gettext('Modified successfully'))
}

onMounted(() => {
  getStatus()
  getIsEnabled()
})
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
      <n-button
        v-if="currentTab == 'error-log'"
        class="ml-16"
        type="primary"
        @click="handleClearErrorLog"
      >
        <the-icon :size="18" icon="material-symbols:delete-outline" />
        {{ $gettext('Clear Log') }}
      </n-button>
      <n-button
        v-if="currentTab == 'slow-log'"
        class="ml-16"
        type="primary"
        @click="handleClearSlowLog"
      >
        <the-icon :size="18" icon="material-symbols:delete-outline" />
        {{ $gettext('Clear Slow Log') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <n-space vertical>
          <n-card :title="$gettext('Running Status')">
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked> {{ $gettext('Autostart On') }} </template>
                <template #unchecked> {{ $gettext('Autostart Off') }} </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="statusType">
                {{ statusStr }}
              </n-alert>
              <n-space>
                <n-button type="success" @click="handleStart">
                  <the-icon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  {{ $gettext('Start') }}
                </n-button>
                <n-popconfirm @positive-click="handleStop">
                  <template #trigger>
                    <n-button type="error">
                      <the-icon :size="24" icon="material-symbols:stop-outline-rounded" />
                      {{ $gettext('Stop') }}
                    </n-button>
                  </template>
                  {{
                    $gettext(
                      'Stopping MySQL will cause websites using MySQL to become inaccessible. Are you sure you want to stop?'
                    )
                  }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <the-icon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
          <n-card :title="$gettext('Root Password')">
            <n-space vertical>
              <n-input
                v-model:value="rootPassword"
                type="password"
                show-password-on="click"
              ></n-input>
              <n-button type="primary" @click="handleSetRootPassword">{{
                $gettext('Save Changes')
              }}</n-button>
            </n-space>
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Modify Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the MySQL main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!'
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
          :scroll-x="400"
          :loading="false"
          :columns="loadColumns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="mysqld" />
      </n-tab-pane>
      <n-tab-pane name="slow-log" :tab="$gettext('Slow Query Log')">
        <realtime-log :path="slowLog" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
