<script setup lang="ts">
defineOptions({
  name: 'apps-minio-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import minio from '@/api/apps/minio'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')

const { data: env } = useRequest(minio.env, {
  initialData: ''
})

const handleSaveEnv = () => {
  useRequest(minio.saveEnv(env.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'env'" class="ml-16" type="primary" @click="handleSaveEnv">
        <the-icon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="minio" />
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
