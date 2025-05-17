<script setup lang="ts">
defineOptions({
  name: 'apps-phpmyadmin-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import phpmyadmin from '@/api/apps/phpmyadmin'

const { $gettext } = useGettext()
const currentTab = ref('status')
const hostname = ref(window.location.hostname)
const port = ref(0)
const path = ref('')
const newPort = ref(0)
const url = computed(() => {
  return `http://${hostname.value}:${port.value}/${path.value}`
})

const { data: config } = useRequest(phpmyadmin.config, {
  initialData: {
    config: ''
  }
})

const getInfo = async () => {
  const data = await phpmyadmin.info()
  path.value = data.path
  port.value = data.port
  newPort.value = data.port
}

const handleSave = () => {
  useRequest(phpmyadmin.port(newPort.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
    getInfo()
  })
}

const handleSaveConfig = () => {
  useRequest(phpmyadmin.updateConfig(config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

onMounted(() => {
  getInfo()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'status'" class="ml-16" type="primary" @click="handleSave">
        <the-icon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
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
      <n-tab-pane name="status" :tab="$gettext('Status')">
        <n-space vertical>
          <n-card :title="$gettext('Access Information')">
            <n-alert type="info">
              {{ $gettext('Access URL:') }} <a :href="url" target="_blank">{{ url }}</a>
            </n-alert>
          </n-card>
          <n-card :title="$gettext('Modify Port')">
            <n-input-number v-model:value="newPort" :min="1" :max="65535" />
            {{ $gettext('Modify phpMyAdmin access port') }}
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Modify Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the OpenResty configuration file for phpMyAdmin. If you do not understand the meaning of each parameter, please do not modify it randomly!'
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
    </n-tabs>
  </common-page>
</template>
