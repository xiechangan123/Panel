<script setup lang="ts">
import { NButton, NDataTable, NFlex, NInput, NPopconfirm, NTag } from 'naive-ui'

import container from '@/api/panel/container'
import { formatDateTime } from '@/utils'
import type { ImageList } from '@/views/container/types'

const pullModel = ref({
  name: '',
  auth: false,
  username: '',
  password: ''
})
const pullModal = ref(false)
const loading = ref(false)

const data = ref<ImageList[]>([] as ImageList[])
const selectedRowKeys = ref<any>([])

const onChecked = (rowKeys: any) => {
  selectedRowKeys.value = rowKeys
}

const columns: any = [
  { type: 'selection', fixed: 'left' },
  {
    title: 'ID',
    key: 'id',
    minWidth: 400,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '容器数',
    key: 'containers',
    width: 100,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '镜像',
    key: 'repo_tags',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true },
    render(row: any): any {
      return h(NFlex, null, {
        default: () =>
          row.repo_tags.map((tag: any) =>
            h(NTag, null, {
              default: () => tag
            })
          )
      })
    }
  },
  {
    title: '大小',
    key: 'size',
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
    width: 120,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NPopconfirm,
          {
            onPositiveClick: async () => {
              await handleDelete(row)
            }
          },
          {
            default: () => {
              return '确定删除吗？'
            },
            trigger: () => {
              return h(
                NButton,
                {
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

const pagination = reactive({
  page: 1,
  pageCount: 1,
  pageSize: 20,
  itemCount: 0,
  showQuickJumper: true,
  showSizePicker: true,
  pageSizes: [20, 50, 100, 200]
})

const onPageChange = (page: number) => {
  pagination.page = page
  getImageList(page, pagination.pageSize).then((res) => {
    data.value = res.items
    pagination.itemCount = res.total
    pagination.pageCount = res.total / pagination.pageSize + 1
  })
}

const onPageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  onPageChange(1)
}

const getImageList = async (page: number, pageSize: number) => {
  const { data } = await container.imageList(page, pageSize)
  return data
}

const handleDelete = async (row: any) => {
  container.imageRemove(row.id).then(() => {
    window.$message.success('删除成功')
    onPageChange(pagination.page)
  })
}

const handlePrune = () => {
  container.imagePrune().then(() => {
    window.$message.success('清理成功')
    onPageChange(pagination.page)
  })
}

const handlePull = () => {
  loading.value = true
  container
    .imagePull(pullModel.value)
    .then(() => {
      window.$message.success('拉取成功')
      onPageChange(pagination.page)
    })
    .finally(() => {
      loading.value = false
      pullModal.value = false
    })
}

onMounted(() => {
  onPageChange(pagination.page)
})
</script>

<template>
  <n-space vertical size="large">
    <n-card rounded-10>
      <n-space>
        <n-button type="primary" @click="pullModal = true">拉取镜像</n-button>
        <n-button type="primary" @click="handlePrune" ghost>清理镜像</n-button>
      </n-space>
    </n-card>
    <n-card rounded-10>
      <n-data-table
        striped
        remote
        :scroll-x="1000"
        :data="data"
        :columns="columns"
        :row-key="(row: any) => row.id"
        :pagination="pagination"
        :bordered="false"
        :loading="false"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
        @update:checked-row-keys="onChecked"
      />
    </n-card>
  </n-space>
  <n-modal
    v-model:show="pullModal"
    preset="card"
    title="拉取镜像"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form :model="pullModel">
      <n-form-item path="name" label="镜像名">
        <n-input
          v-model:value="pullModel.name"
          type="text"
          @keydown.enter.prevent
          placeholder="docker.io/php:8.3-fpm"
        />
      </n-form-item>
      <n-form-item path="auth" label="验证">
        <n-switch v-model:value="pullModel.auth" />
      </n-form-item>
      <n-form-item v-if="pullModel.auth" path="username" label="用户名">
        <n-input
          v-model:value="pullModel.username"
          type="text"
          @keydown.enter.prevent
          placeholder="输入用户名"
        />
      </n-form-item>
      <n-form-item v-if="pullModel.auth" path="password" label="密码">
        <n-input
          v-model:value="pullModel.password"
          type="password"
          show-password-on="click"
          @keydown.enter.prevent
          placeholder="输入密码"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block :loading="loading" :disabled="loading" @click="handlePull">
      提交
    </n-button>
  </n-modal>
</template>
