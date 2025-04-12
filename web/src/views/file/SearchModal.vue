<script setup lang="ts">
import file from '@/api/panel/file'
import { NButton, NPopconfirm, NSpace, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import copy2clipboard from '@vavt/copy2clipboard'
import type { DataTableColumns } from 'naive-ui'
import type { RowData } from 'naive-ui/es/data-table/src/interface'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const path = defineModel<string>('path', { type: String, required: true })
const keyword = defineModel<string>('keyword', { type: String, required: true })
const sub = defineModel<boolean>('sub', { type: Boolean, required: true })

const loading = ref(false)

const columns: DataTableColumns<RowData> = [
  {
    title: $gettext('Name'),
    key: 'full',
    minWidth: 300,
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: $gettext('Size'),
    key: 'size',
    width: 80,
    render(row: any): any {
      return h(NTag, { type: 'info', size: 'small', bordered: false }, { default: () => row.size })
    }
  },
  {
    title: $gettext('Modification Time'),
    key: 'modify',
    width: 200,
    render(row: any): any {
      return h(
        NTag,
        { type: 'warning', size: 'small', bordered: false },
        { default: () => row.modify }
      )
    }
  },
  {
    title: $gettext('Actions'),
    key: 'action',
    width: 200,
    render(row) {
      return h(
        NSpace,
        {},
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                type: 'success',
                tertiary: true,
                onClick: () => {
                  copy2clipboard(row.full).then(() => {
                    window.$message.success($gettext('Copied successfully'))
                  })
                }
              },
              {
                default: () => {
                  return $gettext('Copy Path')
                }
              }
            ),
            h(
              NPopconfirm,
              {
                onPositiveClick: () => {
                  useRequest(file.delete(row.full)).onSuccess(() => {
                    window.$bus.emit('file:refresh')
                    window.$message.success($gettext('Deleted successfully'))
                  })
                },
                onNegativeClick: () => {}
              },
              {
                default: () => {
                  return $gettext('Are you sure you want to delete %{ name }?', { name: row.name })
                },
                trigger: () => {
                  return h(
                    NButton,
                    {
                      size: 'small',
                      type: 'error',
                      tertiary: true
                    },
                    { default: () => $gettext('Delete') }
                  )
                }
              }
            )
          ]
        }
      )
    }
  }
]

const data = ref<RowData[]>([])

const pagination = reactive({
  page: 1,
  pageCount: 1,
  pageSize: 100,
  itemCount: 0,
  showQuickJumper: true,
  showSizePicker: true,
  pageSizes: [100, 200, 500, 1000, 1500, 2000, 5000]
})

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  handlePageChange(1)
}

const handlePageChange = (page: number) => {
  search(page)
}

const search = async (page: number) => {
  loading.value = true
  useRequest(
    file.search(path.value, keyword.value, sub.value, page, pagination.pageSize!)
  ).onSuccess(({ data }) => {
    data.value = data.items
    pagination.itemCount = data.total
    pagination.pageCount = data.total / pagination.pageSize! + 1
  })
  loading.value = false
}

watch(show, (value) => {
  if (value) {
    search(1)
  }
})
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('%{ keyword } - Search Results', { keyword })"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-data-table
      remote
      striped
      virtual-scroll
      size="small"
      :scroll-x="800"
      :columns="columns"
      :data="data"
      :loading="loading"
      :pagination="pagination"
      :row-key="(row: any) => row.full"
      max-height="60vh"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </n-modal>
</template>

<style scoped lang="scss"></style>
