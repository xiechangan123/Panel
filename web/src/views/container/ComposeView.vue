<script setup lang="ts">
import { NButton, NCheckbox, NDataTable, NFlex, NInput, NPopconfirm, NTag } from 'naive-ui'

import container from '@/api/panel/container'
import { useFileStore } from '@/store'
import { formatDateTime } from '@/utils'

const fileStore = useFileStore()
const router = useRouter()

const forcePush = ref(false)

const createModel = ref({
  name: '',
  compose: '',
  envs: []
})
const createModal = ref(false)

const updateModel = ref({
  name: '',
  compose: '',
  envs: []
})
const updateModal = ref(false)

const columns: any = [
  {
    title: '名称',
    key: 'name',
    minWidth: 150,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '目录',
    key: 'path',
    minWidth: 150,
    resizable: true,
    render(row: any) {
      return h(
        NTag,
        {
          class: 'cursor-pointer hover:opacity-60',
          type: 'info',
          onClick: () => {
            fileStore.path = row.path
            router.push({ name: 'file-index' })
          }
        },
        { default: () => row.path }
      )
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 150,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 200,
    resizable: true,
    render(row: any) {
      return formatDateTime(row.created_at)
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 280,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NButton,
          {
            size: 'small',
            onClick: () => {
              useRequest(container.composeGet(row.name)).onSuccess(({ data }: { data: any }) => {
                updateModel.value = {
                  name: row.name,
                  compose: data.compose,
                  envs: data.envs
                }
                updateModal.value = true
              })
            }
          },
          {
            default: () => '编辑'
          }
        ),
        h(
          NPopconfirm,
          {
            showIcon: false,
            onPositiveClick: () => {
              const messageReactive = window.$message.loading('启动中...', {
                duration: 0
              })
              useRequest(container.composeUp(row.name, forcePush.value))
                .onSuccess(() => {
                  refresh()
                  forcePush.value = false
                  window.$message.success('启动成功')
                })
                .onComplete(() => {
                  messageReactive?.destroy()
                })
            }
          },
          {
            default: () => {
              return h(
                NFlex,
                {
                  vertical: true
                },
                {
                  default: () => [
                    h('strong', {}, { default: () => `确定启动编排 ${row.name} 吗？` }),
                    h(
                      NCheckbox,
                      {
                        checked: forcePush.value,
                        onUpdateChecked: (v) => (forcePush.value = v)
                      },
                      { default: () => '强制拉取镜像' }
                    )
                  ]
                }
              )
            },
            trigger: () => {
              return h(
                NButton,
                {
                  style: 'margin-left: 15px;',
                  size: 'small',
                  type: 'success'
                },
                {
                  default: () => '启动'
                }
              )
            }
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => {
              useRequest(container.composeDown(row.name)).onSuccess(() => {
                refresh()
                window.$message.success('停止成功')
              })
            }
          },
          {
            default: () => {
              return `确定停止编排 ${row.name} 吗？`
            },
            trigger: () => {
              return h(
                NButton,
                {
                  style: 'margin-left: 15px;',
                  size: 'small',
                  type: 'warning'
                },
                {
                  default: () => '停止'
                }
              )
            }
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => {
              useRequest(container.composeRemove(row.name)).onSuccess(() => {
                refresh()
                window.$message.success('删除成功')
              })
            }
          },
          {
            default: () => {
              return `确定删除编排 ${row.name} 吗？`
            },
            trigger: () => {
              return h(
                NButton,
                {
                  style: 'margin-left: 15px;',
                  size: 'small',
                  type: 'error'
                },
                {
                  default: () => '删除'
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
  (page, pageSize) => container.composeList(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleCreate = () => {
  loading.value = true
  useRequest(container.composeCreate(createModel.value))
    .onSuccess(() => {
      refresh()
      window.$message.success('创建成功')
    })
    .onComplete(() => {
      loading.value = false
      createModal.value = false
      createModel.value = {
        name: '',
        compose: '',
        envs: []
      }
    })
}

const handleUpdate = () => {
  loading.value = true
  useRequest(container.composeUpdate(updateModel.value.name, updateModel.value))
    .onSuccess(() => {
      refresh()
      window.$message.success('更新成功')
    })
    .onComplete(() => {
      loading.value = false
      updateModal.value = false
      updateModel.value = {
        name: '',
        compose: '',
        envs: []
      }
    })
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <n-flex vertical :size="20">
    <n-flex>
      <n-button type="primary" @click="createModal = true">创建编排</n-button>
    </n-flex>
    <n-data-table
      striped
      remote
      :loading="loading"
      :scroll-x="1000"
      :data="data"
      :columns="columns"
      :row-key="(row: any) => row.id"
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
    v-model:show="createModal"
    preset="card"
    title="创建编排"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form :model="createModel">
      <n-form-item path="name" label="编排名">
        <n-input v-model:value="createModel.name" type="text" />
      </n-form-item>
      <n-form-item path="compose" label="编排">
        <n-input
          v-model:value="createModel.compose"
          type="textarea"
          :autosize="{ minRows: 10, maxRows: 20 }"
        />
      </n-form-item>
      <n-form-item path="envs" label="环境变量">
        <n-dynamic-input
          v-model:value="createModel.envs"
          preset="pair"
          key-placeholder="变量名"
          value-placeholder="变量值"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block :loading="loading" :disabled="loading" @click="handleCreate">
      提交
    </n-button>
  </n-modal>
  <n-modal
    v-model:show="updateModal"
    preset="card"
    title="编辑编排"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form :model="updateModel">
      <n-form-item path="compose" label="编排">
        <n-input
          v-model:value="updateModel.compose"
          type="textarea"
          :autosize="{ minRows: 10, maxRows: 20 }"
        />
      </n-form-item>
      <n-form-item path="envs" label="环境变量">
        <n-dynamic-input
          v-model:value="updateModel.envs"
          preset="pair"
          key-placeholder="变量名"
          value-placeholder="变量值"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block :loading="loading" :disabled="loading" @click="handleUpdate">
      提交
    </n-button>
  </n-modal>
</template>
