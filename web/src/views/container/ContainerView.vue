<script setup lang="ts">
import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NDropdown, NFlex, NInput, NSwitch, NTag } from 'naive-ui'

import container from '@/api/panel/container'
import ContainerCreate from '@/views/container/ContainerCreate.vue'

const logModal = ref(false)
const logs = ref('')
const renameModal = ref(false)
const renameModel = ref({
  id: '',
  name: ''
})

const containerCreateModal = ref(false)
const selectedRowKeys = ref<any>([])

const columns: any = [
  { type: 'selection', fixed: 'left' },
  {
    title: '容器名',
    key: 'name',
    minWidth: 150,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '状态',
    key: 'state',
    width: 100,
    resizable: true,
    render(row: any) {
      return h(NSwitch, {
        size: 'small',
        rubberBand: false,
        value: row.state === 'running',
        onUpdateValue: (value: boolean) => {
          if (value) {
            handleStart(row.id)
          } else {
            handleStop(row.id)
          }
        }
      })
    }
  },
  {
    title: '镜像',
    key: 'image',
    minWidth: 300,
    resizable: true,
    render(row: any): any {
      return h(NTag, null, {
        default: () => row.image
      })
    }
  },
  {
    title: '端口（主机->容器）',
    key: 'ports',
    minWidth: 200,
    resizable: true,
    render(row: any): any {
      return h(NFlex, null, {
        default: () =>
          row.ports.map((port: any) =>
            h(NTag, null, {
              default: () =>
                `${port.host ? port.host + ':' : ''}${port.container_start}->${port.host_start}/${port.protocol}`
            })
          )
      })
    }
  },
  {
    title: '运行状态',
    key: 'status',
    width: 300,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '操作',
    key: 'actions',
    width: 250,
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
            onClick: () => handleShowLog(row)
          },
          {
            default: () => '日志'
          }
        ),
        h(
          NButton,
          {
            size: 'small',
            type: 'success',
            style: 'margin-left: 15px;',
            onClick: () => {
              renameModel.value.id = row.id
              renameModel.value.name = row.name
              renameModal.value = true
            }
          },
          {
            default: () => '重命名'
          }
        ),
        h(
          NDropdown,
          {
            options: [
              {
                label: '启动',
                key: 'start',
                disabled: row.state === 'running'
              },
              {
                label: '停止',
                key: 'stop',
                disabled: row.state !== 'running'
              },
              {
                label: '重启',
                key: 'restart',
                disabled: row.state !== 'running'
              },
              {
                label: '强制停止',
                key: 'forceStop',
                disabled: row.state !== 'running'
              },
              {
                label: '暂停',
                key: 'pause',
                disabled: row.state !== 'running'
              },
              {
                label: '恢复',
                key: 'unpause',
                disabled: row.state === 'running'
              },
              {
                label: '删除',
                key: 'delete'
              }
            ],
            onSelect: (key: string) => {
              switch (key) {
                case 'start':
                  handleStart(row.id)
                  break
                case 'stop':
                  handleStop(row.id)
                  break
                case 'restart':
                  handleRestart(row.id)
                  break
                case 'forceStop':
                  handleForceStop(row.id)
                  break
                case 'pause':
                  handlePause(row.id)
                  break
                case 'unpause':
                  handleUnpause(row.id)
                  break
                case 'delete':
                  handleDelete(row.id)
                  break
              }
            }
          },
          {
            default: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'primary',
                  style: 'margin-left: 15px;'
                },
                {
                  default: () => '更多'
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
  (page, pageSize) => container.containerList(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleShowLog = async (row: any) => {
  useRequest(container.containerLogs(row.id)).onSuccess(({ data }) => {
    logs.value = data
    logModal.value = true
  })
}

const handleRename = () => {
  useRequest(container.containerRename(renameModel.value.id, renameModel.value.name)).onSuccess(
    () => {
      refresh()
      renameModal.value = false
      window.$message.success('重命名成功')
    }
  )
}

const handleStart = (id: string) => {
  useRequest(container.containerStart(id)).onSuccess(() => {
    refresh()
    window.$message.success('启动成功')
  })
}

const handleStop = (id: string) => {
  useRequest(container.containerStop(id)).onSuccess(() => {
    refresh()
    window.$message.success('停止成功')
  })
}

const handleRestart = (id: string) => {
  useRequest(container.containerRestart(id)).onSuccess(() => {
    refresh()
    window.$message.success('重启成功')
  })
}

const handleForceStop = (id: string) => {
  useRequest(container.containerKill(id)).onSuccess(() => {
    refresh()
    window.$message.success('强制停止成功')
  })
}

const handlePause = (id: string) => {
  useRequest(container.containerPause(id)).onSuccess(() => {
    refresh()
    window.$message.success('暂停成功')
  })
}

const handleUnpause = (id: string) => {
  useRequest(container.containerUnpause(id)).onSuccess(() => {
    refresh()
    window.$message.success('恢复成功')
  })
}

const handleDelete = (id: string) => {
  useRequest(container.containerRemove(id)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

const handlePrune = () => {
  useRequest(container.containerPrune()).onSuccess(() => {
    refresh()
    window.$message.success('清理成功')
  })
}

const bulkStart = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要启动的容器')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerStart(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success('启动成功')
}

const bulkStop = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要停止的容器')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerStop(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success('停止成功')
}

const bulkRestart = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要重启的容器')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerRestart(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success('重启成功')
}

const bulkForceStop = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要强制停止的容器')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerKill(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success('强制停止成功')
}

const bulkDelete = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要删除的容器')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerRemove(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success('删除成功')
}

const bulkPause = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要暂停的容器')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerPause(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success('暂停成功')
}

const bulkUnpause = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要恢复的容器')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerUnpause(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success('恢复成功')
}

const closeContainerCreateModal = () => {
  containerCreateModal.value = false
  refresh()
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <n-flex vertical :size="20">
    <n-flex>
      <n-button type="primary" @click="containerCreateModal = true">创建容器</n-button>
      <n-button type="primary" @click="handlePrune" ghost>清理容器</n-button>
      <n-button-group>
        <n-button @click="bulkStart">启动</n-button>
        <n-button @click="bulkStop">停止</n-button>
        <n-button @click="bulkRestart">重启</n-button>
        <n-button @click="bulkForceStop">强制停止</n-button>
        <n-button @click="bulkPause">暂停</n-button>
        <n-button @click="bulkUnpause">恢复</n-button>
        <n-button @click="bulkDelete">删除</n-button>
      </n-button-group>
    </n-flex>
    <n-data-table
      striped
      remote
      :loading="loading"
      :scroll-x="1000"
      :data="data"
      :columns="columns"
      :row-key="(row: any) => row.id"
      v-model:checked-row-keys="selectedRowKeys"
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
  <n-modal
    v-model:show="logModal"
    preset="card"
    title="日志"
    style="width: 80vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <Editor
      v-model:value="logs"
      language="ini"
      theme="vs-dark"
      height="60vh"
      mt-8
      :options="{
        automaticLayout: true,
        formatOnType: true,
        formatOnPaste: true,
        readOnly: true
      }"
    />
  </n-modal>
  <n-modal
    v-model:show="renameModal"
    preset="card"
    title="重命名"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form :model="renameModel">
      <n-form-item path="name" label="新名称">
        <n-input
          v-model:value="renameModel.name"
          type="text"
          @keydown.enter.prevent
          placeholder="输入新名称"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleRename">提交</n-button>
  </n-modal>
  <ContainerCreate :show="containerCreateModal" @close="closeContainerCreateModal" />
</template>
