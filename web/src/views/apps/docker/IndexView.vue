<script setup lang="ts">
defineOptions({
  name: 'apps-docker-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import docker from '@/api/apps/docker'
import systemctl from '@/api/panel/systemctl'

const { $gettext } = useGettext()
const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)

const { data: config } = useRequest(docker.getConfig, {
  initialData: {
    config: ''
  }
})

const statusStr = computed(() => {
  return status.value ? $gettext('Running') : $gettext('Stopped')
})

const getStatus = async () => {
  status.value = await systemctl.status('docker')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('docker')
}

const handleSaveConfig = () => {
  useRequest(docker.updateConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleStart = () => {
  useRequest(systemctl.start('docker')).onSuccess(() => {
    window.$message.success($gettext('Started successfully'))
    getStatus()
  })
}

const handleStop = () => {
  useRequest(systemctl.stop('docker')).onSuccess(() => {
    window.$message.success($gettext('Stopped successfully'))
    getStatus()
  })
}

const handleRestart = () => {
  useRequest(systemctl.restart('docker')).onSuccess(() => {
    window.$message.success($gettext('Restarted successfully'))
    getStatus()
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('docker')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('docker')
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
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
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <n-flex vertical>
          <n-card :title="$gettext('Running Status')">
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked> {{ $gettext('Autostart On') }} </template>
                <template #unchecked> {{ $gettext('Autostart Off') }} </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="status ? 'success' : 'error'">
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
                  {{ $gettext('Are you sure you want to stop Docker?') }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{ $gettext('This modifies the Docker configuration file (/etc/docker/daemon.json)') }}
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
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="docker" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
