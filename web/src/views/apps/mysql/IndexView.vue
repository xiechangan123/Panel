<script setup lang="ts">
defineOptions({
  name: 'apps-mysql-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'

import mysql from '@/api/apps/mysql'
import systemctl from '@/api/panel/systemctl'

const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)

const { data: rootPassword } = useRequest(mysql.rootPassword, {
  initialData: ''
})
const { data: config } = useRequest(mysql.config, {
  initialData: ''
})
const { data: slowLog } = useRequest(mysql.slowLog, {
  initialData: ''
})
const { data: load } = useRequest(mysql.load, {
  initialData: []
})

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? '正常运行中' : '已停止运行'
})

const loadColumns: any = [
  {
    title: '属性',
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '当前值',
    key: 'value',
    minWidth: 200,
    ellipsis: { tooltip: true }
  }
]

const getStatus = async () => {
  status.value = await systemctl.status('mysqld')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('mysqld')
}

const handleSaveConfig = () => {
  useRequest(mysql.saveConfig(config.value)).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const handleClearErrorLog = () => {
  useRequest(mysql.clearErrorLog()).onSuccess(() => {
    window.$message.success('清空成功')
  })
}

const handleClearSlowLog = () => {
  useRequest(mysql.clearSlowLog()).onSuccess(() => {
    window.$message.success('清空成功')
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('mysqld')
    window.$message.success('开启自启动成功')
  } else {
    await systemctl.disable('mysqld')
    window.$message.success('禁用自启动成功')
  }
  await getIsEnabled()
}

const handleStart = async () => {
  await systemctl.start('mysqld')
  window.$message.success('启动成功')
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop('mysqld')
  window.$message.success('停止成功')
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('mysqld')
  window.$message.success('重启成功')
  await getStatus()
}

const handleSetRootPassword = async () => {
  await mysql.setRootPassword(rootPassword.value)
  window.$message.success('修改成功')
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
      <n-button
        v-if="currentTab == 'error-log'"
        class="ml-16"
        type="primary"
        @click="handleClearErrorLog"
      >
        <TheIcon :size="18" icon="material-symbols:delete-outline" />
        清空日志
      </n-button>
      <n-button
        v-if="currentTab == 'slow-log'"
        class="ml-16"
        type="primary"
        @click="handleClearSlowLog"
      >
        <TheIcon :size="18" icon="material-symbols:delete-outline" />
        清空慢日志
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" tab="运行状态">
        <n-space vertical>
          <n-card title="运行状态">
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked> 自启动开 </template>
                <template #unchecked> 自启动关 </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="statusType">
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
                  停止 MySQL 会导致使用 MySQL 的网站无法访问，确定要停止吗？
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  重启
                </n-button>
              </n-space>
            </n-space>
          </n-card>
          <n-card title="Root 密码">
            <n-space vertical>
              <n-input
                v-model:value="rootPassword"
                type="password"
                show-password-on="click"
              ></n-input>
              <n-button type="primary" @click="handleSetRootPassword">保存修改</n-button>
            </n-space>
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="config" tab="修改配置">
        <n-space vertical>
          <n-alert type="warning">
            此处修改的是 MySQL 主配置文件，如果您不了解各参数的含义，请不要随意修改！
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
      <n-tab-pane name="load" tab="负载状态">
        <n-data-table
          striped
          remote
          :scroll-x="400"
          :loading="false"
          :columns="loadColumns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" tab="运行日志">
        <realtime-log service="mysqld" />
      </n-tab-pane>
      <n-tab-pane name="slow-log" tab="慢查询日志">
        <realtime-log :path="slowLog" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
