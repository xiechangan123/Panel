<script setup lang="ts">
import { NButton, NDataTable, NPopconfirm } from 'naive-ui'

import task from '@/api/panel/task'
import RealtimeLogModal from '@/components/common/RealtimeLogModal.vue'
import { formatDateTime, renderIcon } from '@/utils'

const logModal = ref(false)
const logPath = ref('')

const columns: any = [
  {
    title: '任务名',
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '状态',
    key: 'status',
    width: 150,
    ellipsis: { tooltip: true },
    render(row: any) {
      return row.status === 'finished'
        ? '已完成'
        : row.status === 'waiting'
          ? '等待中'
          : row.status === 'failed'
            ? '已失败'
            : '运行中'
    }
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 200,
    ellipsis: { tooltip: true },
    render(row: any): string {
      return formatDateTime(row.created_at)
    }
  },
  {
    title: '完成时间',
    key: 'updated_at',
    width: 200,
    ellipsis: { tooltip: true },
    render(row: any): string {
      return formatDateTime(row.updated_at)
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
        row.status != 'waiting'
          ? h(
              NButton,
              {
                size: 'small',
                type: 'warning',
                secondary: true,
                onClick: () => {
                  logPath.value = row.log
                  logModal.value = true
                }
              },
              {
                default: () => '日志',
                icon: renderIcon('material-symbols:visibility', { size: 14 })
              }
            )
          : null,
        row.status != 'waiting' && row.status != 'running'
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => handleDelete(row.id)
              },
              {
                default: () => {
                  return '确定要删除吗？'
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
          : null
      ]
    }
  }
]

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => task.list(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleDelete = (id: number) => {
  useRequest(task.delete(id)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <n-flex vertical>
    <n-alert type="info">若日志无法加载，请关闭广告拦截应用！</n-alert>
    <n-data-table
      striped
      remote
      :scroll-x="1000"
      :loading="loading"
      :columns="columns"
      :data="data"
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
  <realtime-log-modal v-model:show="logModal" :path="logPath" />
</template>
