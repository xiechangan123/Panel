<script setup lang="ts">
import { NButton, NDataTable, NPopconfirm, NTag } from 'naive-ui'

import firewall from '@/api/panel/firewall'
import { renderIcon } from '@/utils'
import CreateForwardModal from '@/views/firewall/CreateForwardModal.vue'

const createModalShow = ref(false)

const columns: any = [
  { type: 'selection', fixed: 'left' },
  {
    title: '传输协议',
    key: 'protocol',
    width: 150,
    resizable: true,
    ellipsis: { tooltip: true },
    render(row: any): any {
      return h(NTag, null, {
        default: () => {
          if (row.protocol !== '') {
            return row.protocol
          }
          return '无'
        }
      })
    }
  },
  {
    title: '端口',
    key: 'port',
    width: 150,
    render(row: any): any {
      return h(NTag, null, {
        default: () => {
          return row.port
        }
      })
    }
  },
  {
    title: '目标 IP',
    key: 'target_ip',
    minWidth: 200,
    render(row: any): any {
      return h(
        NTag,
        {
          type: 'info'
        },
        {
          default: () => {
            return row.target_ip
          }
        }
      )
    }
  },
  {
    title: '目标端口',
    key: 'target_port',
    width: 150,
    render(row: any): any {
      return h(
        NTag,
        {
          type: 'info'
        },
        {
          default: () => {
            return row.target_port
          }
        }
      )
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
          NPopconfirm,
          {
            onPositiveClick: () => handleDelete(row)
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
      ]
    }
  }
]

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => firewall.forwards(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const selectedRowKeys = ref<any>([])

const handleDelete = (row: any) => {
  useRequest(firewall.deleteForward(row)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

const batchDelete = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要删除的规则')
    return
  }

  const promises = selectedRowKeys.value.map((key: any) => {
    const rule = JSON.parse(key)
    return useRequest(firewall.deleteForward(rule)).onSuccess(() => {
      window.$message.success(`${rule.protocol} ${rule.target_ip}:${rule.target_port} 删除成功`)
    })
  })

  await Promise.all(promises)

  selectedRowKeys.value = []
  await refresh()
}

watch(createModalShow, () => {
  refresh()
})

onMounted(() => {
  refresh()
})
</script>

<template>
  <n-flex vertical :size="20">
    <n-flex items-center>
      <n-button type="primary" @click="createModalShow = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        创建转发
      </n-button>
      <n-popconfirm @positive-click="batchDelete">
        <template #trigger>
          <n-button type="warning">
            <TheIcon :size="18" icon="material-symbols:delete-outline" />
            批量删除
          </n-button>
        </template>
        确定要批量删除吗？
      </n-popconfirm>
    </n-flex>
    <n-data-table
      striped
      remote
      :scroll-x="1000"
      :loading="loading"
      :columns="columns"
      :data="data"
      :row-key="(row: any) => JSON.stringify(row)"
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
  <create-forward-modal v-model:show="createModalShow" />
</template>

<style scoped lang="scss"></style>
