<script setup lang="ts">
import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NDropdown, NFlex, NInput, NSwitch, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import container from '@/api/panel/container'
import ContainerCreate from '@/views/container/ContainerCreate.vue'

const { $gettext } = useGettext()

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
    title: $gettext('Container Name'),
    key: 'name',
    minWidth: 150,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Status'),
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
    title: $gettext('Image'),
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
    title: $gettext('Ports (Host->Container)'),
    key: 'ports',
    minWidth: 200,
    resizable: true,
    render(row: any): any {
      return h(NFlex, null, {
        default: () =>
          row.ports.map((port: any) =>
            h(NTag, null, {
              default: () => {
                if (port.container_start == port.container_end) {
                  return `${port.host ? port.host + ':' : ''}${port.host_start}->${port.container_start}/${port.protocol}`
                }
                return `${port.host ? port.host + ':' : ''}${port.host_start}-${port.host_end}->${port.container_start}-${port.container_end}/${port.protocol}`
              }
            })
          )
      })
    }
  },
  {
    title: $gettext('Running Status'),
    key: 'status',
    width: 300,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 250,
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
            default: () => $gettext('Logs')
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
            default: () => $gettext('Rename')
          }
        ),
        h(
          NDropdown,
          {
            options: [
              {
                label: $gettext('Start'),
                key: 'start',
                disabled: row.state === 'running'
              },
              {
                label: $gettext('Stop'),
                key: 'stop',
                disabled: row.state !== 'running'
              },
              {
                label: $gettext('Restart'),
                key: 'restart',
                disabled: row.state !== 'running'
              },
              {
                label: $gettext('Force Stop'),
                key: 'forceStop',
                disabled: row.state !== 'running'
              },
              {
                label: $gettext('Pause'),
                key: 'pause',
                disabled: row.state !== 'running'
              },
              {
                label: $gettext('Resume'),
                key: 'unpause',
                disabled: row.state === 'running'
              },
              {
                label: $gettext('Delete'),
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
                  default: () => $gettext('More')
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
      window.$message.success($gettext('Rename successful'))
    }
  )
}

const handleStart = (id: string) => {
  useRequest(container.containerStart(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Start successful'))
  })
}

const handleStop = (id: string) => {
  useRequest(container.containerStop(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Stop successful'))
  })
}

const handleRestart = (id: string) => {
  useRequest(container.containerRestart(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Restart successful'))
  })
}

const handleForceStop = (id: string) => {
  useRequest(container.containerKill(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Force stop successful'))
  })
}

const handlePause = (id: string) => {
  useRequest(container.containerPause(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Pause successful'))
  })
}

const handleUnpause = (id: string) => {
  useRequest(container.containerUnpause(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Resume successful'))
  })
}

const handleDelete = (id: string) => {
  useRequest(container.containerRemove(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Delete successful'))
  })
}

const handlePrune = () => {
  useRequest(container.containerPrune()).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Cleanup successful'))
  })
}

const bulkStart = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select containers to start'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerStart(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Start successful'))
}

const bulkStop = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select containers to stop'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerStop(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Stop successful'))
}

const bulkRestart = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select containers to restart'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerRestart(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Restart successful'))
}

const bulkForceStop = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select containers to force stop'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerKill(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Force stop successful'))
}

const bulkDelete = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select containers to delete'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerRemove(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Delete successful'))
}

const bulkPause = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select containers to pause'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerPause(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Pause successful'))
}

const bulkUnpause = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select containers to resume'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => container.containerUnpause(id))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Resume successful'))
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
      <n-button type="primary" @click="containerCreateModal = true">{{
        $gettext('Create Container')
      }}</n-button>
      <n-button type="primary" @click="handlePrune" ghost>{{
        $gettext('Cleanup Containers')
      }}</n-button>
      <n-button-group>
        <n-button @click="bulkStart">{{ $gettext('Start') }}</n-button>
        <n-button @click="bulkStop">{{ $gettext('Stop') }}</n-button>
        <n-button @click="bulkRestart">{{ $gettext('Restart') }}</n-button>
        <n-button @click="bulkForceStop">{{ $gettext('Force Stop') }}</n-button>
        <n-button @click="bulkPause">{{ $gettext('Pause') }}</n-button>
        <n-button @click="bulkUnpause">{{ $gettext('Resume') }}</n-button>
        <n-button @click="bulkDelete">{{ $gettext('Delete') }}</n-button>
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
    :title="$gettext('Logs')"
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
    :title="$gettext('Rename')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form :model="renameModel">
      <n-form-item path="name" :label="$gettext('New Name')">
        <n-input
          v-model:value="renameModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="$gettext('Enter new name')"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleRename">{{ $gettext('Submit') }}</n-button>
  </n-modal>
  <ContainerCreate :show="containerCreateModal" @close="closeContainerCreateModal" />
</template>
