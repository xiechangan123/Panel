<script setup lang="ts">
import { renderIcon } from '@/utils'
import copy2clipboard from '@vavt/copy2clipboard'
import { NButton, NInput, NInputGroup, NPopconfirm, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import database from '@/api/panel/database'
import { formatDateTime } from '@/utils'
import UpdateServerModal from '@/views/database/UpdateServerModal.vue'

const { $gettext } = useGettext()
const updateModal = ref(false)
const updateID = ref(0)

const columns: any = [
  {
    title: $gettext('Type'),
    key: 'type',
    width: 150,
    render(row: any) {
      return h(
        NTag,
        { type: 'info' },
        {
          default: () => {
            switch (row.type) {
              case 'mysql':
                return 'MySQL'
              case 'postgresql':
                return 'PostgreSQL'
              default:
                return row.type
            }
          }
        }
      )
    }
  },
  {
    title: $gettext('Name'),
    key: 'name',
    minWidth: 100,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Username'),
    key: 'username',
    width: 150,
    ellipsis: { tooltip: true },
    render(row: any) {
      return row.username || $gettext('None')
    }
  },
  {
    title: $gettext('Password'),
    key: 'password',
    width: 250,
    render(row: any) {
      return h(NInputGroup, null, {
        default: () => [
          h(NInput, {
            value: row.password,
            type: 'password',
            showPasswordOn: 'click',
            readonly: true,
            placeholder: $gettext('None')
          }),
          h(
            NButton,
            {
              type: 'primary',
              ghost: true,
              onClick: () => {
                copy2clipboard(row.password).then(() => {
                  window.$message.success($gettext('Copied successfully'))
                })
              }
            },
            { default: () => $gettext('Copy') }
          )
        ]
      })
    }
  },
  {
    title: $gettext('Host'),
    key: 'host',
    width: 150,
    render(row: any) {
      return h(NTag, null, {
        default: () => `${row.host}:${row.port}`
      })
    }
  },
  {
    title: $gettext('Comment'),
    key: 'remark',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true },
    render(row: any) {
      return h(NInput, {
        size: 'small',
        value: row.remark,
        onBlur: () => handleRemark(row),
        onUpdateValue(v) {
          row.remark = v
        }
      })
    }
  },
  {
    title: $gettext('Status'),
    key: 'status',
    width: 100,
    render(row: any) {
      return h(
        NTag,
        { type: row.status === 'valid' ? 'success' : 'error' },
        { default: () => (row.status === 'valid' ? $gettext('Valid') : $gettext('Invalid')) }
      )
    }
  },
  {
    title: $gettext('Update Date'),
    key: 'updated_at',
    width: 200,
    ellipsis: { tooltip: true },
    render(row: any) {
      return formatDateTime(row.updated_at)
    }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 300,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NPopconfirm,
          {
            onPositiveClick: () => {
              useRequest(database.serverSync(row.id)).onSuccess(() => {
                refresh()
                window.$message.success($gettext('Synchronized successfully'))
              })
            }
          },
          {
            default: () => {
              return $gettext(
                'Are you sure you want to synchronize database users (excluding password) to the panel?'
              )
            },
            trigger: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'success'
                },
                {
                  default: () => $gettext('Sync'),
                  icon: renderIcon('material-symbols:sync', { size: 14 })
                }
              )
            }
          }
        ),
        h(
          NButton,
          {
            size: 'small',
            type: 'primary',
            style: 'margin-left: 15px;',
            onClick: () => {
              updateID.value = row.id
              updateModal.value = true
            }
          },
          {
            default: () => $gettext('Modify'),
            icon: renderIcon('material-symbols:edit-outline', { size: 14 })
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => {
              // 防手贱
              if (['local_mysql', 'local_postgresql'].includes(row.name)) {
                window.$message.error(
                  $gettext(
                    'Built-in servers cannot be deleted. If you need to delete them, please uninstall the corresponding app'
                  )
                )
                return
              }
              handleDelete(row.id)
            }
          },
          {
            default: () => {
              return $gettext('Are you sure you want to delete the server?')
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
  (page, pageSize) => database.serverList(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleDelete = (id: number) => {
  useRequest(database.serverDelete(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Deleted successfully'))
  })
}

const handleRemark = (row: any) => {
  useRequest(database.serverRemark(row.id, row.remark)).onSuccess(() => {
    window.$message.success($gettext('Modified successfully'))
  })
}

onMounted(() => {
  window.$bus.on('database-server:refresh', () => {
    refresh()
  })
})

onUnmounted(() => {
  window.$bus.off('database-server:refresh')
})
</script>

<template>
  <n-data-table
    striped
    remote
    :scroll-x="1700"
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
  <update-server-modal v-model:id="updateID" v-model:show="updateModal" />
</template>

<style scoped lang="scss"></style>
