<script setup lang="ts">
defineOptions({
  name: 'apps-s3fs-index'
})

import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'

import s3fs from '@/api/apps/s3fs'
import { renderIcon } from '@/utils'

const addMountModal = ref(false)

const addMountModel = ref({
  ak: '',
  sk: '',
  bucket: '',
  url: '',
  path: ''
})

const columns: any = [
  {
    title: '挂载路径',
    key: 'path',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  { title: 'Bucket', key: 'bucket', resizable: true, minWidth: 250, ellipsis: { tooltip: true } },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDeleteMount(row.id)
          },
          {
            default: () => {
              return '确定删除挂载' + row.path + '吗？'
            },
            trigger: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'error'
                },
                {
                  default: () => '卸载',
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
  (page, pageSize) => s3fs.mounts(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleAddMount = async () => {
  useRequest(s3fs.add(addMountModel.value)).onSuccess(() => {
    refresh()
    addMountModal.value = false
    window.$message.success('添加成功')
  })
}

const handleDeleteMount = async (id: number) => {
  useRequest(s3fs.delete(id)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button class="ml-16" type="primary" @click="addMountModal = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        添加挂载
      </n-button>
    </template>
    <n-card title="挂载列表" :segmented="true" rounded-10>
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
    </n-card>
  </common-page>
  <n-modal v-model:show="addMountModal" title="添加挂载">
    <n-card closable @close="() => (addMountModal = false)" title="添加挂载" style="width: 60vw">
      <n-form :model="addMountModel">
        <n-form-item path="bucket" label="Bucket">
          <n-input
            v-model:value="addMountModel.bucket"
            type="text"
            @keydown.enter.prevent
            placeholder="输入 Bucket 名（COS 为: xxxx-ID）"
          />
        </n-form-item>
        <n-form-item path="ak" label="AK">
          <n-input
            v-model:value="addMountModel.ak"
            type="text"
            @keydown.enter.prevent
            placeholder="输入 AK 密钥"
          />
        </n-form-item>
        <n-form-item path="sk" label="SK">
          <n-input
            v-model:value="addMountModel.sk"
            type="text"
            @keydown.enter.prevent
            placeholder="输入 SK 密钥"
          />
        </n-form-item>
        <n-form-item path="url" label="地域节点">
          <n-input
            v-model:value="addMountModel.url"
            type="text"
            @keydown.enter.prevent
            placeholder="输入地域节点的完整 URL（https://oss-cn-beijing.aliyuncs.com）"
          />
        </n-form-item>
        <n-form-item path="path" label="挂载目录">
          <n-input
            v-model:value="addMountModel.path"
            type="text"
            @keydown.enter.prevent
            placeholder="输入挂载目录（/oss）"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleAddMount">提交</n-button>
    </n-card>
  </n-modal>
</template>
