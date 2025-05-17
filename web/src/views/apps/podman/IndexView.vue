<script setup lang="ts">
defineOptions({
  name: 'apps-podman-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import podman from '@/api/apps/podman'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')

const { data: registryConfig } = useRequest(podman.registryConfig, {
  initialData: ''
})

const { data: storageConfig } = useRequest(podman.storageConfig, {
  initialData: ''
})

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
          <service-status service="podman" />
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
