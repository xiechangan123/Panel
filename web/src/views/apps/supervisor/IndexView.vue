<script setup lang="ts">
defineOptions({
  name: 'apps-supervisor-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'

import supervisor from '@/api/apps/supervisor'
import systemctl from '@/api/panel/systemctl'
import { renderIcon } from '@/utils'

const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)
const processLog = ref('')

const { data: serviceName } = useRequest(supervisor.service, {
  initialData: 'supervisor'
}).onSuccess(() => {
  refresh()
  getStatus()
  getIsEnabled()
  config.value = supervisor.config()
})

const { data: config } = useRequest(supervisor.config, {
  initialData: ''
})

const createProcessModal = ref(false)
const createProcessModel = ref({
  name: '',
  user: 'www',
  path: '',
  command: '',
  num: 1
})

const editProcessModal = ref(false)
const editProcessModel = ref({
  process: '',
  config: ''
})

const processLogModal = ref(false)

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? '正常运行中' : '已停止运行'
})

const processColumns: any = [
  {
    title: '名称',
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '状态',
    key: 'status',
    minWidth: 100,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: 'PID',
    key: 'pid',
    minWidth: 100,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '运行时间',
    key: 'uptime',
    minWidth: 150,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '操作',
    key: 'actions',
    width: 500,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NButton,
          {
            size: 'small',
            type: 'warning',
            secondary: true,
            onClick: () => handleShowProcessLog(row)
          },
          {
            default: () => '日志',
            icon: renderIcon('material-symbols:visibility', { size: 14 })
          }
        ),
        h(
          NButton,
          {
            size: 'small',
            type: 'info',
            style: 'margin-left: 15px',
            onClick: () => handleEditProcess(row.name)
          },
          {
            default: () => '配置',
            icon: renderIcon('material-symbols:settings-outline', { size: 14 })
          }
        ),
        row.status != 'RUNNING'
          ? h(
              NButton,
              {
                size: 'small',
                type: 'primary',
                secondary: true,
                style: 'margin-left: 15px',
                onClick: () => handleProcessStart(row.name)
              },
              {
                default: () => '启动',
                icon: renderIcon('material-symbols:play-arrow-outline', { size: 18 })
              }
            )
          : null,
        row.status == 'RUNNING'
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => handleProcessStop(row.name)
              },
              {
                default: () => {
                  return '确定停止进程' + row.name + '吗？'
                },
                trigger: () => {
                  return h(
                    NButton,
                    {
                      size: 'small',
                      type: 'warning',
                      style: 'margin-left: 15px'
                    },
                    {
                      default: () => '停止',
                      icon: renderIcon('material-symbols:stop-outline', { size: 18 })
                    }
                  )
                }
              }
            )
          : null,
        row.status == 'RUNNING'
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => handleProcessRestart(row.name)
              },
              {
                default: () => {
                  return '确定重启进程' + row.name + '吗？'
                },
                trigger: () => {
                  return h(
                    NButton,
                    {
                      size: 'small',
                      type: 'primary',
                      style: 'margin-left: 15px'
                    },
                    {
                      default: () => '重启',
                      icon: renderIcon('material-symbols:replay', { size: 18 })
                    }
                  )
                }
              }
            )
          : null,
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleProcessDelete(row.name)
          },
          {
            default: () => {
              return '确定删除进程' + row.name + '吗？'
            },
            trigger: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'error',
                  style: 'margin-left: 15px'
                },
                {
                  default: () => '删除',
                  icon: renderIcon('material-symbols:delete-outline', { size: 14 })
                }
              )
            }
          }
        )
      ]
    }
  }
]

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => supervisor.processes(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const getStatus = async () => {
  status.value = await systemctl.status(serviceName.value)
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled(serviceName.value)
}

const handleSaveConfig = async () => {
  useRequest(supervisor.saveConfig(config.value)).onSuccess(() => {
    refresh()
    window.$message.success('保存成功')
  })
}

const handleClearLog = async () => {
  useRequest(supervisor.clearLog()).onSuccess(() => {
    window.$message.success('清空成功')
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    useRequest(systemctl.enable(serviceName.value)).onSuccess(() => {
      getIsEnabled()
      window.$message.success('开启自启动成功')
    })
  } else {
    useRequest(systemctl.disable(serviceName.value)).onSuccess(() => {
      getIsEnabled()
      window.$message.success('禁用自启动成功')
    })
  }
}

const handleStart = async () => {
  useRequest(systemctl.start(serviceName.value)).onSuccess(() => {
    getStatus()
    window.$message.success('启动成功')
  })
}

const handleStop = async () => {
  useRequest(systemctl.stop(serviceName.value)).onSuccess(() => {
    getStatus()
    window.$message.success('停止成功')
  })
}

const handleRestart = async () => {
  useRequest(systemctl.restart(serviceName.value)).onSuccess(() => {
    getStatus()
    window.$message.success('重启成功')
  })
}

const handleCreateProcess = async () => {
  useRequest(supervisor.createProcess(createProcessModel.value)).onSuccess(() => {
    refresh()
    createProcessModal.value = false
    window.$message.success('添加成功')
  })
}

const handleProcessStart = async (name: string) => {
  useRequest(supervisor.startProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success('启动成功')
  })
}

const handleProcessStop = async (name: string) => {
  useRequest(supervisor.stopProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success('停止成功')
  })
}

const handleProcessRestart = async (name: string) => {
  useRequest(supervisor.restartProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success('重启成功')
  })
}

const handleProcessDelete = async (name: string) => {
  useRequest(supervisor.deleteProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

const handleShowProcessLog = async (row: any) => {
  processLog.value = await supervisor.processLog(row.name)
  processLogModal.value = true
}

const handleEditProcess = async (name: string) => {
  await getProcessConfig(name)
  editProcessModal.value = true
}

const getProcessConfig = async (name: string) => {
  editProcessModel.value.process = name
  editProcessModel.value.config = await supervisor.processConfig(name)
}

const handleSaveProcessConfig = async () => {
  useRequest(
    supervisor.saveProcessConfig(editProcessModel.value.process, editProcessModel.value.config)
  ).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const timer: any = null

onUnmounted(() => {
  clearInterval(timer)
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
        v-if="currentTab == 'processes'"
        class="ml-16"
        type="primary"
        @click="createProcessModal = true"
      >
        <TheIcon :size="18" icon="material-symbols:add" />
        添加进程
      </n-button>
      <n-button v-if="currentTab == 'log'" class="ml-16" type="primary" @click="handleClearLog">
        <TheIcon :size="18" icon="material-symbols:delete-outline" />
        清空日志
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" tab="运行状态">
        <n-space vertical>
          <n-card title="运行状态" rounded-10>
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked> 自启动开</template>
                <template #unchecked> 自启动关</template>
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
                  停止 Supervisor 会导致 Supervisor 管理的所有进程被杀死，确定要停止吗？
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  重启
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="processes" tab="进程管理">
        <n-card title="进程列表" :segmented="true" rounded-10>
          <n-data-table
            striped
            remote
            :scroll-x="1000"
            :loading="loading"
            :columns="processColumns"
            :data="data"
            :row-key="(row: any) => row.name"
            v-model:page="page"
            v-model:pageSize="pageSize"
            :pagination="{
              page: page,
              pageCount: pageCount,
              pageSize: pageSize,
              itemCount: total,
              showQuickJumper: true,
              showSizePicker: true,
              pageSizes: [20, 50, 100, 200]
            }"
          />
        </n-card>
      </n-tab-pane>
      <n-tab-pane name="config" tab="主配置">
        <n-space vertical>
          <n-alert type="warning">
            此处修改的是 Supervisor 主配置文件，如果您不了解各参数的含义，请不要随意修改！
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
      <n-tab-pane name="log" tab="日志">
        <realtime-log path="/var/log/supervisor/supervisord.log" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
  <n-modal
    v-model:show="createProcessModal"
    preset="card"
    title="添加进程"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="createProcessModal = false"
  >
    <n-form :model="createProcessModel">
      <n-form-item path="name" label="名称">
        <n-input
          v-model:value="createProcessModel.name"
          type="text"
          @keydown.enter.prevent
          placeholder="名称禁止使用中文"
        />
      </n-form-item>
      <n-form-item path="command" label="启动命令">
        <n-input
          v-model:value="createProcessModel.command"
          type="text"
          @keydown.enter.prevent
          placeholder="启动命令中的文件请填写绝对路径"
        />
      </n-form-item>
      <n-form-item path="path" label="运行目录">
        <n-input
          v-model:value="createProcessModel.path"
          type="text"
          @keydown.enter.prevent
          placeholder="运行目录请填写绝对路径"
        />
      </n-form-item>
      <n-form-item path="user" label="启动用户">
        <n-input
          v-model:value="createProcessModel.user"
          type="text"
          @keydown.enter.prevent
          placeholder="一般情况下填写www即可"
        />
      </n-form-item>
      <n-form-item path="num" label="进程数量">
        <n-input-number v-model:value="createProcessModel.num" :min="1" />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleCreateProcess">提交</n-button>
  </n-modal>
  <realtime-log-modal v-model:show="processLogModal" :path="processLog" />
  <n-modal
    v-model:show="editProcessModal"
    preset="card"
    title="进程配置"
    style="width: 80vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="handleSaveProcessConfig"
  >
    <Editor
      v-model:value="editProcessModel.config"
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
  </n-modal>
</template>
