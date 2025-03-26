<script setup lang="ts">
import backup from '@/api/panel/backup'
import { renderIcon } from '@/utils'
import type { MessageReactive } from 'naive-ui'
import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'

import app from '@/api/panel/app'
import website from '@/api/panel/website'
import { formatDateTime } from '@/utils'

const type = defineModel<string>('type', { type: String, required: true })

let messageReactive: MessageReactive | null = null

const restoreModal = ref(false)
const restoreModel = ref({
  file: '',
  target: ''
})

const websites = ref<any>([])

const columns: any = [
  {
    title: '文件名',
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '大小',
    key: 'size',
    width: 160,
    ellipsis: { tooltip: true }
  },
  {
    title: '更新日期',
    key: 'time',
    width: 200,
    ellipsis: { tooltip: true },
    render(row: any) {
      return formatDateTime(row.time)
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 200,
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
            onClick: () => {
              restoreModel.value.file = row.path
              restoreModal.value = true
            }
          },
          {
            default: () => '恢复',
            icon: renderIcon('material-symbols:settings-backup-restore-rounded', { size: 14 })
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDelete(row.name)
          },
          {
            default: () => {
              return '确定删除备份吗？'
            },
            trigger: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'error',
                  style: 'margin-left: 15px;'
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
  (page, pageSize) => backup.list(type.value, page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleRestore = () => {
  messageReactive = window.$message.loading('恢复中...', {
    duration: 0
  })

  useRequest(backup.restore(type.value, restoreModel.value.file, restoreModel.value.target))
    .onSuccess(() => {
      refresh()
      window.$message.success('恢复成功')
    })
    .onComplete(() => {
      messageReactive?.destroy()
    })
}

const handleDelete = async (file: string) => {
  useRequest(backup.delete(type.value, file)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

onMounted(() => {
  useRequest(app.isInstalled('nginx')).onSuccess(({ data }) => {
    if (data.installed) {
      useRequest(website.list(1, 10000)).onSuccess(({ data }: { data: any }) => {
        for (const item of data.items) {
          websites.value.push({
            label: item.name,
            value: item.name
          })
        }
        if (type.value === 'website') {
          restoreModel.value.target = websites.value[0]?.value
        }
      })
    }
  })
  refresh()
  window.$bus.on('backup:refresh', () => {
    refresh()
  })
})

onUnmounted(() => {
  window.$bus.off('backup:refresh')
})
</script>

<template>
  <n-data-table
    striped
    remote
    :scroll-x="1000"
    :loading="loading"
    :columns="columns"
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
  <n-modal
    v-model:show="restoreModal"
    preset="card"
    title="恢复备份"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="restoreModal = false"
  >
    <n-form :model="restoreModel">
      <n-form-item v-if="type == 'website'" path="name" label="网站">
        <n-select v-model:value="restoreModel.target" :options="websites" placeholder="选择网站" />
      </n-form-item>
      <n-form-item v-if="type != 'website'" path="name" label="数据库">
        <n-input v-model:value="restoreModel.target" type="text" @keydown.enter.prevent />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleRestore">提交</n-button>
  </n-modal>
</template>

<style scoped lang="scss"></style>
