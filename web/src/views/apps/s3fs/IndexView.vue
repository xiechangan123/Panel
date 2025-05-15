<script setup lang="ts">
defineOptions({
  name: 'apps-s3fs-index'
})

import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import s3fs from '@/api/apps/s3fs'
import { renderIcon } from '@/utils'

const { $gettext } = useGettext()
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
    title: $gettext('Mount Path'),
    key: 'path',
    minWidth: 150,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  { title: 'Bucket', key: 'bucket', resizable: true, minWidth: 150, ellipsis: { tooltip: true } },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 150,
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
              return $gettext('Are you sure you want to delete mount %{ path }?', {
                path: row.path
              })
            },
            trigger: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'error'
                },
                {
                  default: () => $gettext('Unmount'),
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

const handleAddMount = () => {
  useRequest(s3fs.add(addMountModel.value)).onSuccess(() => {
    refresh()
    addMountModal.value = false
    window.$message.success($gettext('Added successfully'))
  })
}

const handleDeleteMount = (id: number) => {
  useRequest(s3fs.delete(id)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Deleted successfully'))
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
        <the-icon :size="18" icon="material-symbols:add" />
        {{ $gettext('Add Mount') }}
      </n-button>
    </template>
    <n-flex vertical>
      <n-data-table
        striped
        remote
        :scroll-x="450"
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
  </common-page>
  <n-modal v-model:show="addMountModal" :title="$gettext('Add Mount')">
    <n-card
      closable
      @close="() => (addMountModal = false)"
      :title="$gettext('Add Mount')"
      style="width: 60vw"
    >
      <n-form :model="addMountModel">
        <n-form-item path="bucket" label="Bucket">
          <n-input
            v-model:value="addMountModel.bucket"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter Bucket name (COS format: xxxx-ID)')"
          />
        </n-form-item>
        <n-form-item path="ak" label="AK">
          <n-input
            v-model:value="addMountModel.ak"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter AK key')"
          />
        </n-form-item>
        <n-form-item path="sk" label="SK">
          <n-input
            v-model:value="addMountModel.sk"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter SK key')"
          />
        </n-form-item>
        <n-form-item path="url" :label="$gettext('Region Endpoint')">
          <n-input
            v-model:value="addMountModel.url"
            type="text"
            @keydown.enter.prevent
            :placeholder="
              $gettext(
                'Enter complete URL of region endpoint (e.g., https://oss-cn-beijing.aliyuncs.com)'
              )
            "
          />
        </n-form-item>
        <n-form-item path="path" :label="$gettext('Mount Directory')">
          <n-input
            v-model:value="addMountModel.path"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter mount directory (e.g., /oss)')"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleAddMount">{{ $gettext('Submit') }}</n-button>
    </n-card>
  </n-modal>
</template>
