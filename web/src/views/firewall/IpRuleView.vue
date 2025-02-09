<script setup lang="ts">
import { NButton, NDataTable, NPopconfirm, NTag } from 'naive-ui'

import firewall from '@/api/panel/firewall'
import { renderIcon } from '@/utils'
import CreateIpModal from '@/views/firewall/CreateIpModal.vue'

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
    title: '网络协议',
    key: 'family',
    width: 150,
    resizable: true,
    ellipsis: { tooltip: true },
    render(row: any): any {
      return h(NTag, null, {
        default: () => {
          if (row.family !== '') {
            return row.family
          }
          return '无'
        }
      })
    }
  },
  {
    title: '策略',
    key: 'strategy',
    width: 150,
    render(row: any): any {
      return h(
        NTag,
        {
          type:
            row.strategy === 'accept'
              ? 'success'
              : row.strategy === 'drop'
                ? 'warning'
                : row.strategy === 'reject'
                  ? 'error'
                  : 'default'
        },
        {
          default: () => {
            switch (row.strategy) {
              case 'accept':
                return '接受'
              case 'drop':
                return '丢弃'
              case 'reject':
                return '拒绝'
              case 'mark':
                return '标记'
              default:
                return '未知'
            }
          }
        }
      )
    }
  },
  {
    title: '方向',
    key: 'direction',
    width: 150,
    render(row: any): any {
      return h(
        NTag,
        {
          type: row.direction === 'in' ? 'info' : 'default'
        },
        {
          default: () => {
            switch (row.direction) {
              case 'in':
                return '传入'
              case 'out':
                return '传出'
              default:
                return '未知'
            }
          }
        }
      )
    }
  },
  {
    title: '目标',
    key: 'address',
    minWidth: 200,
    render(row: any): any {
      return h(NTag, null, {
        default: () => {
          return row.address
        }
      })
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
  (page, pageSize) => firewall.ipRules(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const selectedRowKeys = ref<any>([])

const handleDelete = (row: any) => {
  useRequest(firewall.deleteIpRule(row)).onSuccess(() => {
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
    return useRequest(firewall.deleteIpRule(rule)).onSuccess(() => {
      window.$message.success(`${rule.address} 删除成功`)
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
        创建规则
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
  <create-ip-modal v-model:show="createModalShow" />
</template>

<style scoped lang="scss"></style>
