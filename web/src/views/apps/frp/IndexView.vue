<script setup lang="ts">
defineOptions({
  name: 'apps-frp-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import frp from '@/api/apps/frp'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('frps')
const config = ref({
  frpc: '',
  frps: ''
})

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

onMounted(() => {
  getConfig()
})
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="frps" tab="Frps">
        <n-flex vertical>
          <service-status service="frps" />
          <n-card :title="$gettext('Modify Configuration')">
            <template #header-extra>
              <n-button type="primary" @click="handleSaveConfig('frps')">
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
                smoothScrolling: true
              }"
            />
          </n-card>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="frpc" tab="Frpc">
        <n-flex vertical>
          <service-status service="frpc" />
          <n-card :title="$gettext('Modify Configuration')">
            <template #header-extra>
              <n-button type="primary" @click="handleSaveConfig('frpc')">
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
                smoothScrolling: true
              }"
            />
          </n-card>
        </n-flex>
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
