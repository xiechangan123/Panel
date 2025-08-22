<script setup lang="ts">
import { NButton, NDataTable, NPopconfirm, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import firewall from '@/api/panel/firewall'
import CreateModal from '@/views/firewall/CreateModal.vue'

const { $gettext } = useGettext()
const createModalShow = ref(false)

const columns: any = [
  { type: 'selection', fixed: 'left' },
  {
    title: $gettext('Transport Protocol'),
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
          return $gettext('None')
        }
      })
    }
  },
  {
    title: $gettext('Network Protocol'),
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
          return $gettext('None')
        }
      })
    }
  },
  {
    title: $gettext('Port'),
    key: 'port',
    width: 250,
    resizable: true,
    ellipsis: { tooltip: true },
    render(row: any): any {
      if (row.port_start == row.port_end) {
        return row.port_start
      }
      return `${row.port_start}-${row.port_end}`
    }
  },
  {
    title: $gettext('Status'),
    key: 'in_use',
    width: 150,
    render(row: any): any {
      return h(
        NTag,
        {
          type: row.in_use ? 'success' : 'default'
        },
        {
          default: () => {
            if (row.in_use) {
              return $gettext('In Use')
            }
            return $gettext('Not Used')
          }
        }
      )
    }
  },
  {
    title: $gettext('Strategy'),
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
                return $gettext('Accept')
              case 'drop':
                return $gettext('Drop')
              case 'reject':
                return $gettext('Reject')
              case 'mark':
                return $gettext('Mark')
              default:
                return $gettext('Unknown')
            }
          }
        }
      )
    }
  },
  {
    title: $gettext('Direction'),
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
                return $gettext('Inbound')
              case 'out':
                return $gettext('Outbound')
              default:
                return $gettext('Unknown')
            }
          }
        }
      )
    }
  },
  {
    title: $gettext('Target'),
    key: 'address',
    minWidth: 200,
    render(row: any): any {
      return h(NTag, null, {
        default: () => {
          if (row.address === '') {
            return $gettext('All')
          }
          return row.address
        }
      })
    }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 200,
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
              return $gettext('Are you sure you want to delete?')
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
                  default: () => $gettext('Delete')
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
  (page, pageSize) => firewall.rules(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const selectedRowKeys = ref<any>([])

const handleDelete = async (row: any) => {
  useRequest(firewall.deleteRule(row)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Deleted successfully'))
  })
}

const batchDelete = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select rules to delete'))
    return
  }

  const promises = selectedRowKeys.value.map((key: any) => {
    const rule = JSON.parse(key)
    return firewall.deleteRule(rule)
  })
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Deleted successfully'))
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
        {{ $gettext('Create Rule') }}
      </n-button>
      <n-popconfirm @positive-click="batchDelete">
        <template #trigger>
          <n-button type="warning">
            {{ $gettext('Batch Delete') }}
          </n-button>
        </template>
        {{ $gettext('Are you sure you want to batch delete?') }}
      </n-popconfirm>
    </n-flex>
    <n-data-table
      striped
      remote
      :scroll-x="1400"
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
  <create-modal v-model:show="createModalShow" />
</template>

<style scoped lang="scss"></style>
