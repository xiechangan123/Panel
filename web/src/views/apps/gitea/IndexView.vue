<script setup lang="ts">
defineOptions({
  name: 'apps-gitea-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import gitea from '@/api/apps/gitea'
import systemctl from '@/api/panel/systemctl'

const { $gettext } = useGettext()
const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)
const config = ref('')

const statusStr = computed(() => {
  return status.value ? $gettext('Running') : $gettext('Stopped')
})

const getStatus = async () => {
  status.value = await systemctl.status('gitea')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('gitea')
}

const getConfig = async () => {
  config.value = await gitea.config()
}

const handleSaveConfig = () => {
  useRequest(gitea.saveConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleStart = async () => {
  await systemctl.start('gitea')
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop('gitea')
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('gitea')
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('gitea')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('gitea')
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
}

onMounted(() => {
  getStatus()
  getIsEnabled()
  getConfig()
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
                {{ $gettext('Are you sure you want to stop Gitea?') }}
              </n-popconfirm>
              <n-button type="warning" @click="handleRestart">
                <the-icon :size="18" icon="material-symbols:replay-rounded" />
                {{ $gettext('Restart') }}
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
                'This modifies the Gitea configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!'
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
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="gitea" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
