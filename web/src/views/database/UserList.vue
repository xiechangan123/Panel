<script setup lang="ts">
import { renderIcon } from '@/utils'
import copy2clipboard from '@vavt/copy2clipboard'
import { NButton, NFlex, NInput, NInputGroup, NPopconfirm, NTag } from 'naive-ui'

import database from '@/api/panel/database'
import { formatDateTime } from '@/utils'
import UpdateUserModal from '@/views/database/UpdateUserModal.vue'

const updateModal = ref(false)
const updateID = ref(0)

const columns: any = [
  {
    title: '类型',
    key: 'type',
    width: 150,
    render(row: any) {
      return h(
        NTag,
        { type: 'info' },
        {
          default: () => {
            switch (row.server.type) {
              case 'mysql':
                return 'MySQL'
              case 'postgresql':
                return 'PostgreSQL'
              default:
                return row.server.type
            }
          }
        }
      )
    }
  },
  {
    title: '用户名',
    key: 'username',
    minWidth: 100,
    resizable: true,
    ellipsis: { tooltip: true },
    render(row: any) {
      return row.username || '无'
    }
  },
  {
    title: '密码',
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
            placeholder: '未保存'
          }),
          h(
            NButton,
            {
              type: 'primary',
              ghost: true,
              onClick: () => {
                copy2clipboard(row.password).then(() => {
                  window.$message.success('复制成功')
                })
              }
            },
            { default: () => '复制' }
          )
        ]
      })
    }
  },
  {
    title: '主机',
    key: 'host',
    width: 150,
    render(row: any) {
      return h(NTag, null, {
        default: () => row.host || '无'
      })
    }
  },
  {
    title: '服务器',
    key: 'server',
    width: 150,
    render(row: any) {
      return row.server.name
    }
  },
  {
    title: '授权',
    key: 'privileges',
    width: 200,
    render(row: any) {
      return h(NFlex, null, {
        default: () =>
          row.privileges.map((privilege: string) =>
            h(NTag, null, {
              default: () => privilege
            })
          )
      })
    }
  },
  {
    title: '备注',
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
    title: '状态',
    key: 'status',
    width: 100,
    render(row: any) {
      return h(
        NTag,
        { type: row.status === 'valid' ? 'success' : 'error' },
        { default: () => (row.status === 'valid' ? '有效' : '无效') }
      )
    }
  },
  {
    title: '更新日期',
    key: 'updated_at',
    width: 200,
    ellipsis: { tooltip: true },
    render(row: any) {
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
        h(
          NButton,
          {
            size: 'small',
            type: 'primary',
            onClick: () => {
              updateID.value = row.id
              updateModal.value = true
            }
          },
          {
            default: () => '修改',
            icon: renderIcon('material-symbols:edit-outline', { size: 14 })
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDelete(row.id)
          },
          {
            default: () => {
              return '确定删除用户吗？'
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
  (page, pageSize) => database.userList(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleDelete = (id: number) => {
  useRequest(database.userDelete(id)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

const handleRemark = (row: any) => {
  useRequest(database.userRemark(row.id, row.remark)).onSuccess(() => {
    window.$message.success('修改成功')
  })
}

onMounted(() => {
  window.$bus.on('database-user:refresh', () => {
    refresh()
  })
})

onUnmounted(() => {
  window.$bus.off('database-user:refresh')
})
</script>

<template>
  <n-data-table
    striped
    remote
    :scroll-x="1800"
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
  <update-user-modal v-model:id="updateID" v-model:show="updateModal" />
</template>

<style scoped lang="scss"></style>
