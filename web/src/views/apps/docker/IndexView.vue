<script setup lang="ts">
defineOptions({
  name: 'apps-docker-index'
})

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
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="docker" />
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Configuration')">
        <n-flex vertical>
          <n-alert type="warning">
            {{ $gettext('This modifies the Docker configuration file (/etc/docker/daemon.json)') }}
          </n-alert>
          <common-editor v-model:value="config" height="60vh" />
          <n-flex>
            <n-button type="primary" @click="handleSaveConfig">
              {{ $gettext('Save') }}
            </n-button>
          </n-flex>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="docker" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
