<script setup lang="ts">
defineOptions({
  name: 'apps-minio-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import minio from '@/api/apps/minio'
import systemctl from '@/api/panel/systemctl'

const { $gettext } = useGettext()
const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)
const env = ref('')

const statusStr = computed(() => {
  return status.value ? $gettext('Running normally') : $gettext('Stopped')
})

const getStatus = async () => {
  status.value = await systemctl.status('minio')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('minio')
}

const getEnv = async () => {
  env.value = await minio.env()
}

const handleSaveEnv = () => {
  useRequest(minio.saveEnv(env.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleStart = async () => {
  await systemctl.start('minio')
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop('minio')
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('minio')
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('minio')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('minio')
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
}

onMounted(() => {
  getStatus()
  getIsEnabled()
  getEnv()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'env'" class="ml-16" type="primary" @click="handleSaveEnv">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
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
                {{ $gettext('Are you sure you want to stop Minio?') }}
              </n-popconfirm>
              <n-button type="warning" @click="handleRestart">
                <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                {{ $gettext('Restart') }}
              </n-button>
            </n-space>
          </n-space>
        </n-card>
      </n-tab-pane>
      <n-tab-pane name="env" :tab="$gettext('Environment Variables')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This is modifying the Minio environment variable file /etc/default/minio. If you do not understand the meaning of each parameter, please do not modify it arbitrarily!'
              )
            }}
          </n-alert>
          <Editor
            v-model:value="env"
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
        <realtime-log service="minio" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
