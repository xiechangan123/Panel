<script setup lang="ts">
defineOptions({
  name: 'apps-caddy-index',
})

import { useGettext } from 'vue3-gettext'

import caddy from '@/api/apps/caddy'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')
const saveConfigLoading = ref(false)
const clearErrorLogLoading = ref(false)

const { data: config } = useRequest(caddy.config, {
  initialData: '',
})
const { data: errorLog } = useRequest(caddy.errorLog, {
  initialData: '',
})

const handleSaveConfig = () => {
  saveConfigLoading.value = true
  useRequest(caddy.saveConfig(config.value))
    .onSuccess(() => {
      window.$message.success($gettext('Saved successfully'))
    })
    .onComplete(() => {
      saveConfigLoading.value = false
    })
}

const errorLogRef = ref<{ clear: () => void } | null>(null)

const handleClearErrorLog = () => {
  clearErrorLogLoading.value = true
  useRequest(caddy.clearErrorLog())
    .onSuccess(() => {
      errorLogRef.value?.clear()
      window.$message.success($gettext('Cleared successfully'))
    })
    .onComplete(() => {
      clearErrorLogLoading.value = false
    })
}
</script>

<template>
  <PageContainer :show-footer="true">
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="caddy" show-reload />
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Modify Configuration')">
        <n-flex vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the %{name} main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!',
                { name: 'Caddy' },
              )
            }}
          </n-alert>
          <common-editor v-model:value="config" lang="plaintext" height="60vh" />
          <n-flex>
            <n-button
              type="primary"
              :loading="saveConfigLoading"
              :disabled="saveConfigLoading"
              @click="handleSaveConfig"
            >
              {{ $gettext('Save') }}
            </n-button>
          </n-flex>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="caddy" />
      </n-tab-pane>
      <n-tab-pane name="error-log" :tab="$gettext('Error Logs')">
        <n-flex vertical>
          <n-flex>
            <n-button
              type="primary"
              :loading="clearErrorLogLoading"
              :disabled="clearErrorLogLoading"
              @click="handleClearErrorLog"
            >
              {{ $gettext('Clear Log') }}
            </n-button>
          </n-flex>
          <realtime-log ref="errorLogRef" :path="errorLog" />
        </n-flex>
      </n-tab-pane>
    </n-tabs>
  </PageContainer>
</template>
