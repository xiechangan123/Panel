<script setup lang="ts">
defineOptions({
  name: 'apps-nginx-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import nginx from '@/api/apps/nginx'
import systemctl from '@/api/panel/systemctl'

const { $gettext } = useGettext()
const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)

const { data: config } = useRequest(nginx.config, {
  initialData: ''
})
const { data: errorLog } = useRequest(nginx.errorLog, {
  initialData: ''
})
const { data: load } = useRequest(nginx.load, {
  initialData: []
})

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})

const statusStr = computed(() => {
  return status.value ? $gettext('Running') : $gettext('Stopped')
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

const getStatus = async () => {
  status.value = await systemctl.status('nginx')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('nginx')
}

const handleSaveConfig = () => {
  useRequest(nginx.saveConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleClearErrorLog = () => {
  useRequest(nginx.clearErrorLog()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('nginx')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('nginx')
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
}

const handleStart = async () => {
  await systemctl.start('nginx')
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop('nginx')
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('nginx')
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleReload = async () => {
  await systemctl.reload('nginx')
  window.$message.success($gettext('Reloaded successfully'))
  await getStatus()
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
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button
        v-if="currentTab == 'error-log'"
        class="ml-16"
        type="primary"
        @click="handleClearErrorLog"
      >
        <TheIcon :size="18" icon="material-symbols:delete-outline" />
        {{ $gettext('Clear Log') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
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
                <TheIcon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                {{ $gettext('Start') }}
              </n-button>
              <n-popconfirm @positive-click="handleStop">
                <template #trigger>
                  <n-button type="error">
                    <TheIcon :size="24" icon="material-symbols:stop-outline-rounded" />
                    {{ $gettext('Stop') }}
                  </n-button>
                </template>
                {{
                  $gettext(
                    'Stopping OpenResty will cause all websites to become inaccessible. Are you sure you want to stop?'
                  )
                }}
              </n-popconfirm>
              <n-button type="warning" @click="handleRestart">
                <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                {{ $gettext('Restart') }}
              </n-button>
              <n-button type="primary" @click="handleReload">
                <TheIcon :size="20" icon="material-symbols:refresh-rounded" />
                {{ $gettext('Reload') }}
              </n-button>
            </n-space>
          </n-space>
        </n-card>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Modify Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the OpenResty main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!'
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
          :columns="columns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="nginx" />
      </n-tab-pane>
      <n-tab-pane name="error-log" :tab="$gettext('Error Logs')">
        <realtime-log :path="errorLog" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
