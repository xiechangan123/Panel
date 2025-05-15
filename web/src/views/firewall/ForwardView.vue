<script setup lang="ts">
import { NButton, NDataTable, NPopconfirm, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import firewall from '@/api/panel/firewall'
import { renderIcon } from '@/utils'
import CreateForwardModal from '@/views/firewall/CreateForwardModal.vue'

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
    title: $gettext('Port'),
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
    title: $gettext('Target IP'),
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
    title: $gettext('Target Port'),
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
    return firewall.deleteForward(rule)
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
        <the-icon :size="18" icon="material-symbols:add" />
        {{ $gettext('Create Forwarding') }}
      </n-button>
      <n-popconfirm @positive-click="batchDelete">
        <template #trigger>
          <n-button type="warning">
            <the-icon :size="18" icon="material-symbols:delete-outline" />
            {{ $gettext('Batch Delete') }}
          </n-button>
        </template>
        {{ $gettext('Are you sure you want to batch delete?') }}
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
