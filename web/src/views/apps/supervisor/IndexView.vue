<script setup lang="ts">
defineOptions({
  name: 'apps-supervisor-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import supervisor from '@/api/apps/supervisor'
import systemctl from '@/api/panel/systemctl'
import { renderIcon } from '@/utils'

const { $gettext } = useGettext()
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
  return status.value ? $gettext('Running') : $gettext('Stopped')
})

const processColumns: any = [
  {
    title: $gettext('Name'),
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Status'),
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
    title: $gettext('Uptime'),
    key: 'uptime',
    minWidth: 150,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 500,
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
            default: () => $gettext('Logs'),
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
            default: () => $gettext('Configure'),
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
                default: () => $gettext('Start'),
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
                  return $gettext('Are you sure you want to stop process %{ name }?', {
                    name: row.name
                  })
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
                      default: () => $gettext('Stop'),
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
                  return $gettext('Are you sure you want to restart process %{ name }?', {
                    name: row.name
                  })
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
                      default: () => $gettext('Restart'),
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
              return $gettext('Are you sure you want to delete process %{ name }?', {
                name: row.name
              })
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
                  default: () => $gettext('Delete'),
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
    initialPageSize: 20,
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

const handleSaveConfig = () => {
  useRequest(supervisor.saveConfig(config.value)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleClearLog = () => {
  useRequest(supervisor.clearLog()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleIsEnabled = () => {
  if (isEnabled.value) {
    useRequest(systemctl.enable(serviceName.value)).onSuccess(() => {
      getIsEnabled()
      window.$message.success($gettext('Autostart enabled successfully'))
    })
  } else {
    useRequest(systemctl.disable(serviceName.value)).onSuccess(() => {
      getIsEnabled()
      window.$message.success($gettext('Autostart disabled successfully'))
    })
  }
}

const handleStart = () => {
  useRequest(systemctl.start(serviceName.value)).onSuccess(() => {
    getStatus()
    window.$message.success($gettext('Started successfully'))
  })
}

const handleStop = () => {
  useRequest(systemctl.stop(serviceName.value)).onSuccess(() => {
    getStatus()
    window.$message.success($gettext('Stopped successfully'))
  })
}

const handleRestart = () => {
  useRequest(systemctl.restart(serviceName.value)).onSuccess(() => {
    getStatus()
    window.$message.success($gettext('Restarted successfully'))
  })
}

const handleCreateProcess = () => {
  useRequest(supervisor.createProcess(createProcessModel.value)).onSuccess(() => {
    refresh()
    createProcessModal.value = false
    window.$message.success($gettext('Added successfully'))
  })
}

const handleProcessStart = (name: string) => {
  useRequest(supervisor.startProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Started successfully'))
  })
}

const handleProcessStop = (name: string) => {
  useRequest(supervisor.stopProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Stopped successfully'))
  })
}

const handleProcessRestart = (name: string) => {
  useRequest(supervisor.restartProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Restarted successfully'))
  })
}

const handleProcessDelete = (name: string) => {
  useRequest(supervisor.deleteProcess(name)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Deleted successfully'))
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

const handleSaveProcessConfig = () => {
  useRequest(
    supervisor.saveProcessConfig(editProcessModel.value.process, editProcessModel.value.config)
  ).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
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
        <the-icon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button
        v-if="currentTab == 'processes'"
        class="ml-16"
        type="primary"
        @click="createProcessModal = true"
      >
        <the-icon :size="18" icon="material-symbols:add" />
        {{ $gettext('Add Process') }}
      </n-button>
      <n-button v-if="currentTab == 'log'" class="ml-16" type="primary" @click="handleClearLog">
        <the-icon :size="18" icon="material-symbols:delete-outline" />
        {{ $gettext('Clear Log') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <n-space vertical>
          <n-card :title="$gettext('Running Status')">
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked>{{ $gettext('Autostart On') }}</template>
                <template #unchecked>{{ $gettext('Autostart Off') }}</template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="statusType">
                {{ statusStr }}
              </n-alert>
              <n-space>
                <n-button type="success" @click="handleStart">
                  <the-icon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  {{ $gettext('Start') }}
                </n-button>
                <n-popconfirm @positive-click="handleStop">
                  <template #trigger>
                    <n-button type="error">
                      <the-icon :size="24" icon="material-symbols:stop-outline-rounded" />
                      {{ $gettext('Stop') }}
                    </n-button>
                  </template>
                  {{
                    $gettext(
                      'Stopping Supervisor will cause all processes managed by Supervisor to be killed. Are you sure you want to stop?'
                    )
                  }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <the-icon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="processes" :tab="$gettext('Process Management')">
        <n-flex vertical>
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
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Main Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the Supervisor main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!'
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
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="supervisor" />
      </n-tab-pane>
      <n-tab-pane name="log" :tab="$gettext('Daemon Logs')">
        <realtime-log path="/var/log/supervisor/supervisord.log" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
  <n-modal
    v-model:show="createProcessModal"
    preset="card"
    :title="$gettext('Add Process')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="createProcessModal = false"
  >
    <n-form :model="createProcessModel">
      <n-form-item path="name" :label="$gettext('Name')">
        <n-input
          v-model:value="createProcessModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="$gettext('Name cannot contain Chinese characters')"
        />
      </n-form-item>
      <n-form-item path="command" :label="$gettext('Start Command')">
        <n-input
          v-model:value="createProcessModel.command"
          type="text"
          @keydown.enter.prevent
          :placeholder="$gettext('Please enter absolute path for files in start command')"
        />
      </n-form-item>
      <n-form-item path="path" :label="$gettext('Working Directory')">
        <n-input
          v-model:value="createProcessModel.path"
          type="text"
          @keydown.enter.prevent
          :placeholder="$gettext('Please enter absolute path for working directory')"
        />
      </n-form-item>
      <n-form-item path="user" :label="$gettext('Run As User')">
        <n-input
          v-model:value="createProcessModel.user"
          type="text"
          @keydown.enter.prevent
          :placeholder="$gettext('Usually www is sufficient')"
        />
      </n-form-item>
      <n-form-item path="num" :label="$gettext('Number of Processes')">
        <n-input-number v-model:value="createProcessModel.num" :min="1" />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleCreateProcess">{{ $gettext('Submit') }}</n-button>
  </n-modal>
  <realtime-log-modal v-model:show="processLogModal" :path="processLog" />
  <n-modal
    v-model:show="editProcessModal"
    preset="card"
    :title="$gettext('Process Configuration')"
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
