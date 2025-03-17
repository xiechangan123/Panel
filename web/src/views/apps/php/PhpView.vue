<script setup lang="ts">
import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NPopconfirm } from 'naive-ui'

import php from '@/api/apps/php'
import systemctl from '@/api/panel/systemctl'
import { renderIcon } from '@/utils'

const props = defineProps({
  version: {
    type: Number,
    required: true
  }
})

const { version } = toRefs(props)

const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)

const { data: config } = useRequest(php.config(version.value), {
  initialData: ''
})
const { data: fpmConfig } = useRequest(php.fpmConfig(version.value), {
  initialData: ''
})
const { data: errorLog } = useRequest(php.errorLog(version.value), {
  initialData: ''
})
const { data: slowLog } = useRequest(php.slowLog(version.value), {
  initialData: ''
})
const { data: load } = useRequest(php.load(version.value), {
  initialData: []
})
const { data: extensions } = useRequest(php.extensions(version.value), {
  initialData: []
})

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? '正常运行中' : '已停止运行'
})

const extensionColumns: any = [
  {
    title: '拓展名',
    key: 'name',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '描述',
    key: 'description',
    resizable: true,
    minWidth: 250,
    ellipsis: { tooltip: true }
  },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        !row.installed
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => handleInstallExtension(row.slug)
              },
              {
                default: () => {
                  return '确定安装 ' + row.name + ' 吗？'
                },
                trigger: () => {
                  return h(
                    NButton,
                    {
                      size: 'small',
                      type: 'info'
                    },
                    {
                      default: () => '安装',
                      icon: renderIcon('material-symbols:download-rounded', { size: 14 })
                    }
                  )
                }
              }
            )
          : null,
        row.installed
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => handleUninstallExtension(row.slug)
              },
              {
                default: () => {
                  return '确定卸载 ' + row.name + ' 吗？'
                },
                trigger: () => {
                  return h(
                    NButton,
                    {
                      size: 'small',
                      type: 'error'
                    },
                    {
                      default: () => '删除',
                      icon: renderIcon('material-symbols:delete-outline', { size: 14 })
                    }
                  )
                }
              }
            )
          : null
      ]
    }
  }
]

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
  status.value = await systemctl.status(`php-fpm-${version.value}`)
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled(`php-fpm-${version.value}`)
}

const handleSetCli = async () => {
  useRequest(php.setCli(version.value)).onSuccess(() => {
    window.$message.success('设置成功')
  })
}

const handleSaveConfig = async () => {
  useRequest(php.saveConfig(version.value, config.value)).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const handleSaveFPMConfig = async () => {
  useRequest(php.saveFPMConfig(version.value, fpmConfig.value)).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const handleClearErrorLog = async () => {
  useRequest(php.clearErrorLog(version.value)).onSuccess(() => {
    window.$message.success('清空成功')
  })
}

const handleClearSlowLog = async () => {
  useRequest(php.clearSlowLog(version.value)).onSuccess(() => {
    window.$message.success('清空成功')
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable(`php-fpm-${version.value}`)
    window.$message.success('开启自启动成功')
  } else {
    await systemctl.disable(`php-fpm-${version.value}`)
    window.$message.success('禁用自启动成功')
  }
  await getIsEnabled()
}

const handleStart = async () => {
  await systemctl.start(`php-fpm-${version.value}`)
  window.$message.success('启动成功')
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop(`php-fpm-${version.value}`)
  window.$message.success('停止成功')
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart(`php-fpm-${version.value}`)
  window.$message.success('重启成功')
  await getStatus()
}

const handleReload = async () => {
  await systemctl.reload(`php-fpm-${version.value}`)
  window.$message.success('重载成功')
  await getStatus()
}

const handleInstallExtension = async (slug: string) => {
  useRequest(php.installExtension(version.value, slug)).onSuccess(() => {
    window.$message.success('任务已提交，请前往后台任务查看进度')
  })
}

const handleUninstallExtension = async (name: string) => {
  useRequest(php.uninstallExtension(version.value, name)).onSuccess(() => {
    window.$message.success('任务已提交，请前往后台任务查看进度')
  })
}

onMounted(() => {
  getStatus()
  getIsEnabled()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'status'" class="ml-16" type="info" @click="handleSetCli">
        设为 CLI 默认版本
      </n-button>
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
        v-if="currentTab == 'fpm-config'"
        class="ml-16"
        type="primary"
        @click="handleSaveFPMConfig"
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
        清空错误日志
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
                  停止 PHP {{ version }} 会导致使用 PHP {{ version }} 的网站无法访问，确定要停止吗？
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  重启
                </n-button>
                <n-button type="primary" @click="handleReload">
                  <TheIcon :size="20" icon="material-symbols:refresh-rounded" />
                  重载
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="extensions" tab="拓展管理">
        <n-card title="拓展列表" :segmented="true">
          <n-data-table
            striped
            remote
            :scroll-x="1000"
            :loading="false"
            :columns="extensionColumns"
            :data="extensions"
            :row-key="(row: any) => row.slug"
          />
        </n-card>
      </n-tab-pane>
      <n-tab-pane name="config" tab="主配置">
        <n-space vertical>
          <n-alert type="warning">
            此处修改的是 PHP {{ version }} 主配置文件，如果您不了解各参数的含义，请不要随意修改！
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
      <n-tab-pane name="fpm-config" tab="FPM 配置">
        <n-space vertical>
          <n-alert type="warning">
            此处修改的是 PHP {{ version }} FPM 配置文件，如果您不了解各参数的含义，请不要随意修改！
          </n-alert>
          <Editor
            v-model:value="fpmConfig"
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
        <realtime-log :service="'php-fpm-' + version" />
      </n-tab-pane>
      <n-tab-pane name="error-log" tab="错误日志">
        <realtime-log :path="errorLog" />
      </n-tab-pane>
      <n-tab-pane name="slow-log" tab="慢日志">
        <realtime-log :path="slowLog" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
