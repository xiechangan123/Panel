<script setup lang="ts">
defineOptions({
  name: 'apps-podman-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import podman from '@/api/apps/podman'
import systemctl from '@/api/panel/systemctl'

const { $gettext } = useGettext()
const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)
const registryConfig = ref('')
const storageConfig = ref('')

const statusStr = computed(() => {
  return status.value ? $gettext('Running') : $gettext('Stopped')
})

const getStatus = async () => {
  status.value = await systemctl.status('podman')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('podman')
}

const getConfig = async () => {
  registryConfig.value = await podman.registryConfig()
  storageConfig.value = await podman.storageConfig()
}

const handleSaveRegistryConfig = () => {
  useRequest(podman.saveRegistryConfig(registryConfig.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleSaveStorageConfig = () => {
  useRequest(podman.saveStorageConfig(storageConfig.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleStart = async () => {
  await systemctl.start('podman')
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop('podman')
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('podman')
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('podman')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('podman')
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
        v-if="currentTab == 'registryConfig'"
        class="ml-16"
        type="primary"
        @click="handleSaveRegistryConfig"
      >
        <the-icon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button
        v-if="currentTab == 'storageConfig'"
        class="ml-16"
        type="primary"
        @click="handleSaveStorageConfig"
      >
        <the-icon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <n-flex vertical>
          <n-alert type="info">
            {{
              $gettext(
                'Podman is a daemonless container management tool. Being in a stopped state is normal and does not affect usage!'
              )
            }}
          </n-alert>
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
                  {{ $gettext('Are you sure you want to stop Podman?') }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <the-icon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="registryConfig" :tab="$gettext('Registry Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the Podman registry configuration file (/etc/containers/registries.conf)'
              )
            }}
          </n-alert>
          <Editor
            v-model:value="registryConfig"
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
      <n-tab-pane name="storageConfig" :tab="$gettext('Storage Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the Podman storage configuration file (/etc/containers/storage.conf)'
              )
            }}
          </n-alert>
          <Editor
            v-model:value="storageConfig"
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
        <realtime-log service="podman" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
