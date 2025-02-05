<script setup lang="ts">
defineOptions({
  name: 'apps-docker-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NPopconfirm } from 'naive-ui'

import docker from '@/api/apps/docker'
import systemctl from '@/api/panel/systemctl'

const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)

const { data: config } = useRequest(docker.getConfig, {
  initialData: {
    config: ''
  }
})

const statusStr = computed(() => {
  return status.value ? '正常运行中' : '已停止运行'
})

const getStatus = async () => {
  status.value = await systemctl.status('docker')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('docker')
}

const handleSaveConfig = async () => {
  useRequest(docker.updateConfig(config.value)).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const handleStart = () => {
  useRequest(systemctl.start('docker')).onSuccess(() => {
    window.$message.success('启动成功')
    getStatus()
  })
}

const handleStop = () => {
  useRequest(systemctl.stop('docker')).onSuccess(() => {
    window.$message.success('停止成功')
    getStatus()
  })
}

const handleRestart = () => {
  useRequest(systemctl.restart('docker')).onSuccess(() => {
    window.$message.success('重启成功')
    getStatus()
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('docker')
    window.$message.success('开启自启动成功')
  } else {
    await systemctl.disable('docker')
    window.$message.success('禁用自启动成功')
  }
  await getIsEnabled()
}

onMounted(() => {
  getStatus()
  getIsEnabled()
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
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        保存
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" tab="运行状态">
        <n-flex vertical>
          <n-card title="运行状态" rounded-10>
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked> 自启动开 </template>
                <template #unchecked> 自启动关 </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="status ? 'success' : 'error'">
                {{ statusStr }}
              </n-alert>
              <n-space>
                <n-button type="success" @click="handleStart">
                  <TheIcon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  启动
                </n-button>
                <n-popconfirm @positive-click="handleStop">
                  <template #trigger>
                    <n-button type="error">
                      <TheIcon :size="24" icon="material-symbols:stop-outline-rounded" />
                      停止
                    </n-button>
                  </template>
                  确定要停止 Docker 吗？
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  重启
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="config" tab="配置">
        <n-space vertical>
          <n-alert type="warning">
            此处修改的是 Docker 配置文件（/etc/docker/daemon.json）
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
