<script setup lang="ts">
defineOptions({
  name: 'apps-docker-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import docker from '@/api/apps/docker'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')

const { data: config } = useRequest(docker.config, {
  initialData: {
    config: ''
  }
})

const handleSaveConfig = () => {
  useRequest(docker.updateConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}
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
        {{ $gettext('Save') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="docker" />
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
