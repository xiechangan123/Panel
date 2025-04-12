<script setup lang="ts">
defineOptions({
  name: 'apps-frp-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import frp from '@/api/apps/frp'
import systemctl from '@/api/panel/systemctl'

const { $gettext } = useGettext()
const currentTab = ref('frps')
const status = ref({
  frpc: false,
  frps: false
})
const isEnabled = ref({
  frpc: false,
  frps: false
})
const config = ref({
  frpc: '',
  frps: ''
})

const statusStr = computed(() => {
  return {
    frpc: status.value.frpc ? $gettext('Running') : $gettext('Stopped'),
    frps: status.value.frps ? $gettext('Running') : $gettext('Stopped')
  }
})

const getStatus = async () => {
  status.value.frps = await systemctl.status('frps')
  status.value.frpc = await systemctl.status('frpc')
}

const getIsEnabled = async () => {
  isEnabled.value.frps = await systemctl.isEnabled('frps')
  isEnabled.value.frpc = await systemctl.isEnabled('frpc')
}

const getConfig = async () => {
  config.value.frps = await frp.config('frps')
  config.value.frpc = await frp.config('frpc')
}

const handleSaveConfig = (service: string) => {
  useRequest(frp.saveConfig(service, config.value[service as keyof typeof config.value])).onSuccess(
    () => {
      window.$message.success($gettext('Saved successfully'))
    }
  )
}

const handleStart = async (name: string) => {
  await systemctl.start(name)
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleStop = async (name: string) => {
  await systemctl.stop(name)
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async (name: string) => {
  await systemctl.restart(name)
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleIsEnabled = async (name: string) => {
  if (isEnabled.value[name as keyof typeof isEnabled.value]) {
    await systemctl.enable(name)
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable(name)
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
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="frps" tab="Frps">
        <n-space vertical>
          <n-card :title="$gettext('Running Status')">
            <template #header-extra>
              <n-switch v-model:value="isEnabled.frps" @update:value="handleIsEnabled('frps')">
                <template #checked> {{ $gettext('Autostart On') }} </template>
                <template #unchecked> {{ $gettext('Autostart Off') }} </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="status.frps ? 'success' : 'error'">
                {{ statusStr.frps }}
              </n-alert>
              <n-space>
                <n-button type="success" @click="handleStart('frps')">
                  <TheIcon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  {{ $gettext('Start') }}
                </n-button>
                <n-popconfirm @positive-click="handleStop('frps')">
                  <template #trigger>
                    <n-button type="error">
                      <TheIcon :size="24" icon="material-symbols:stop-outline-rounded" />
                      {{ $gettext('Stop') }}
                    </n-button>
                  </template>
                  {{ $gettext('Are you sure you want to stop Frps?') }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart('frps')">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
          <n-card :title="$gettext('Modify Configuration')">
            <template #header-extra>
              <n-button type="primary" @click="handleSaveConfig('frps')">
                <TheIcon :size="18" icon="material-symbols:save-outline-rounded" />
                {{ $gettext('Save') }}
              </n-button>
            </template>
            <Editor
              v-model:value="config.frps"
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
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="frpc" tab="Frpc">
        <n-space vertical>
          <n-card :title="$gettext('Running Status')">
            <template #header-extra>
              <n-switch v-model:value="isEnabled.frpc" @update:value="handleIsEnabled('frpc')">
                <template #checked> {{ $gettext('Autostart On') }} </template>
                <template #unchecked> {{ $gettext('Autostart Off') }} </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="status.frpc ? 'success' : 'error'">
                {{ statusStr.frpc }}
              </n-alert>
              <n-space>
                <n-button type="success" @click="handleStart('frpc')">
                  <TheIcon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  {{ $gettext('Start') }}
                </n-button>
                <n-popconfirm @positive-click="handleStop('frpc')">
                  <template #trigger>
                    <n-button type="error">
                      <TheIcon :size="24" icon="material-symbols:stop-outline-rounded" />
                      {{ $gettext('Stop') }}
                    </n-button>
                  </template>
                  {{ $gettext('Are you sure you want to stop Frpc?') }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart('frpc')">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
          <n-card :title="$gettext('Modify Configuration')">
            <template #header-extra>
              <n-button type="primary" @click="handleSaveConfig('frpc')">
                <TheIcon :size="18" icon="material-symbols:save-outline-rounded" />
                {{ $gettext('Save') }}
              </n-button>
            </template>
            <Editor
              v-model:value="config.frpc"
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
          </n-card>
        </n-space>
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
