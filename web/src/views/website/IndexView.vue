<script lang="ts" setup>
defineOptions({
  name: 'website-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NCheckbox, NDataTable, NFlex, NInput, NPopconfirm, NSwitch, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import dashboard from '@/api/panel/dashboard'
import website from '@/api/panel/website'
import { useFileStore } from '@/store'
import { generateRandomString, isNullOrUndef, renderIcon } from '@/utils'
import BulkCreate from '@/views/website/BulkCreate.vue'

const fileStore = useFileStore()
const { $gettext } = useGettext()
const router = useRouter()
const selectedRowKeys = ref<any>([])

const columns: any = [
  { type: 'selection', fixed: 'left' },
  {
    title: $gettext('Website Name'),
    key: 'name',
    width: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Running'),
    key: 'status',
    width: 150,
    render(row: any) {
      return h(NSwitch, {
        size: 'small',
        rubberBand: false,
        value: row.status,
        onUpdateValue: () => handleStatusChange(row)
      })
    }
  },
  {
    title: $gettext('Directory'),
    key: 'path',
    minWidth: 200,
    resizable: true,
    render(row: any) {
      return h(
        NTag,
        {
          class: 'cursor-pointer hover:opacity-60',
          type: 'info',
          onClick: () => {
            fileStore.path = row.path
            router.push({ name: 'file-index' })
          }
        },
        { default: () => row.path }
      )
    }
  },
  {
    title: 'HTTPS',
    key: 'https',
    width: 150,
    render(row: any) {
      return h(NSwitch, {
        size: 'small',
        rubberBand: false,
        value: row.https,
        onClick: () => handleEdit(row)
      })
    }
  },
  {
    title: $gettext('Remark'),
    key: 'remark',
    minWidth: 200,
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
    title: $gettext('Actions'),
    key: 'actions',
    width: 220,
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NButton,
          {
            size: 'small',
            type: 'primary',
            style: 'margin-left: 15px;',
            onClick: () => handleEdit(row)
          },
          {
            default: () => $gettext('Edit'),
            icon: renderIcon('material-symbols:edit-outline', { size: 14 })
          }
        ),
        h(
          NPopconfirm,
          {
            showIcon: false,
            onPositiveClick: () => handleDelete(row.id)
          },
          {
            default: () => {
              return h(
                NFlex,
                {
                  vertical: true
                },
                {
                  default: () => [
                    h(
                      'strong',
                      {},
                      {
                        default: () =>
                          $gettext('Are you sure you want to delete website %{ name }?', {
                            name: row.name
                          })
                      }
                    ),
                    h(
                      NCheckbox,
                      {
                        checked: deleteModel.value.path,
                        onUpdateChecked: (v) => (deleteModel.value.path = v)
                      },
                      { default: () => $gettext('Delete website directory') }
                    ),
                    h(
                      NCheckbox,
                      {
                        checked: deleteModel.value.db,
                        onUpdateChecked: (v) => (deleteModel.value.db = v)
                      },
                      { default: () => $gettext('Delete local database with the same name') }
                    )
                  ]
                }
              )
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

const createModal = ref(false)
const bulkCreateModal = ref(false)
const editDefaultPageModal = ref(false)

const createModel = ref({
  name: '',
  listens: [] as Array<string>,
  domains: [] as Array<string>,
  path: '',
  php: 0,
  db: false,
  db_type: '0',
  db_name: '',
  db_user: '',
  db_password: '',
  remark: ''
})
const deleteModel = ref({
  path: true,
  db: false
})
const editDefaultPageModel = ref({
  index: '',
  stop: ''
})

const { data: installedDbAndPhp } = useRequest(dashboard.installedDbAndPhp, {
  initialData: {
    php: [
      {
        label: $gettext('Not used'),
        value: 0
      }
    ],
    db: [
      {
        label: '',
        value: ''
      }
    ]
  }
})

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => website.list(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

// 修改运行状态
const handleStatusChange = (row: any) => {
  if (isNullOrUndef(row.id)) return

  useRequest(website.status(row.id, !row.status)).onSuccess(() => {
    row.status = !row.status
    if (row.status) {
      window.$message.success($gettext('Started successfully'))
    } else {
      window.$message.success($gettext('Stopped successfully'))
    }
  })
}

const getDefaultPage = async () => {
  editDefaultPageModel.value = await website.defaultConfig()
}

const handleRemark = (row: any) => {
  useRequest(website.updateRemark(row.id, row.remark)).onSuccess(() => {
    window.$message.success($gettext('Modified successfully'))
  })
}

const handleEdit = (row: any) => {
  router.push({
    name: 'website-edit',
    params: {
      id: row.id
    }
  })
}

const handleDelete = (id: number) => {
  useRequest(website.delete(id, deleteModel.value.path, deleteModel.value.db)).onSuccess(() => {
    refresh()
    deleteModel.value.path = true
    window.$message.success($gettext('Deleted successfully'))
  })
}

const handleSaveDefaultPage = () => {
  useRequest(
    website.saveDefaultConfig(editDefaultPageModel.value.index, editDefaultPageModel.value.stop)
  ).onSuccess(() => {
    editDefaultPageModal.value = false
    window.$message.success($gettext('Modified successfully'))
  })
}

const handleCreate = async () => {
  // 去除空的域名和端口
  createModel.value.domains = createModel.value.domains.filter((item) => item !== '')
  createModel.value.listens = createModel.value.listens.filter((item) => item !== '')
  // 端口为空自动添加 80 端口
  if (createModel.value.listens.length === 0) {
    createModel.value.listens.push('80')
  }
  // 端口中去掉 443 端口，nginx 不允许在未配置证书下监听 443 端口
  createModel.value.listens = createModel.value.listens.filter((item) => item !== '443')
  useRequest(website.create(createModel.value)).onSuccess(() => {
    refresh()
    window.$message.success(
      $gettext('Website %{ name } created successfully', { name: createModal.value.name })
    )
    createModal.value = false
    createModel.value = {
      name: '',
      domains: [] as Array<string>,
      listens: [] as Array<string>,
      php: 0,
      db: false,
      db_type: '0',
      db_name: '',
      db_user: '',
      db_password: '',
      path: '',
      remark: ''
    }
  })
}

const bulkDelete = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info($gettext('Please select the websites to delete'))
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => website.delete(id, true, false))
  await Promise.all(promises)

  selectedRowKeys.value = []
  refresh()
  window.$message.success($gettext('Deleted successfully'))
}

const formatDbValue = (value: string) => {
  value = value.replace(/\./g, '_')
  value = value.replace(/-/g, '_')
  if (value.length > 16) {
    value = value.substring(0, 16)
  }

  return value
}

onMounted(() => {
  refresh()
  getDefaultPage()
  window.$bus.on('website:refresh', refresh)
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-flex>
        <n-button type="warning" @click="editDefaultPageModal = true">
          {{ $gettext('Modify Default Page') }}
        </n-button>
        <n-popconfirm @positive-click="bulkDelete">
          <template #trigger>
            <n-button type="error"> {{ $gettext('Batch Delete') }} </n-button>
          </template>
          {{
            $gettext(
              'This will delete the website directory but not the database with the same name. Are you sure you want to delete the selected websites?'
            )
          }}
        </n-popconfirm>
        <n-button type="primary" @click="bulkCreateModal = true">
          {{ $gettext('Bulk Create Website') }}
        </n-button>
        <n-button type="primary" @click="createModal = true">
          {{ $gettext('Create Website') }}
        </n-button>
      </n-flex>
    </template>
    <n-flex vertical :size="20">
      <n-data-table
        striped
        remote
        :loading="loading"
        :scroll-x="1200"
        :columns="columns"
        :data="data"
        :row-key="(row: any) => row.id"
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
  </common-page>
  <n-modal
    v-model:show="createModal"
    :title="$gettext('Create Website')"
    preset="card"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="createModal = false"
  >
    <n-form :model="createModel">
      <n-form-item path="name" :label="$gettext('Website Name')">
        <n-input
          v-model:value="createModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="
            $gettext(
              'Recommended to use English for the website name, it cannot be modified after setting'
            )
          "
        />
      </n-form-item>
      <n-row :gutter="[0, 24]">
        <n-col :span="11">
          <n-form-item :label="$gettext('Domain')">
            <n-dynamic-input
              v-model:value="createModel.domains"
              placeholder="example.com"
              :min="1"
              show-sort-button
            />
          </n-form-item>
        </n-col>
        <n-col :span="2"></n-col>
        <n-col :span="11">
          <n-form-item :label="$gettext('Port')">
            <n-dynamic-input
              v-model:value="createModel.listens"
              placeholder="80"
              :min="1"
              show-sort-button
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-row :gutter="[0, 24]">
        <n-col :span="11">
          <n-form-item path="php" :label="$gettext('PHP Version')">
            <n-select
              v-model:value="createModel.php"
              :options="installedDbAndPhp.php"
              :placeholder="$gettext('Select PHP Version')"
              @keydown.enter.prevent
            >
            </n-select>
          </n-form-item>
        </n-col>
        <n-col :span="2"></n-col>
        <n-col :span="11">
          <n-form-item path="db" :label="$gettext('Database')">
            <n-select
              v-model:value="createModel.db_type"
              :options="installedDbAndPhp.db"
              :placeholder="$gettext('Select Database')"
              @keydown.enter.prevent
              @update:value="
                () => {
                  createModel.db = createModel.db_type != '0'
                  createModel.db_name = formatDbValue(createModel.name)
                  createModel.db_user = formatDbValue(createModel.name)
                  createModel.db_password = generateRandomString(16)
                }
              "
            >
            </n-select>
          </n-form-item>
        </n-col>
      </n-row>
      <n-row :gutter="[0, 24]">
        <n-col :span="7">
          <n-form-item v-if="createModel.db" path="db_name" :label="$gettext('Database Name')">
            <n-input
              v-model:value="createModel.db_name"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Database Name')"
            />
          </n-form-item>
        </n-col>
        <n-col :span="1"></n-col>
        <n-col :span="7">
          <n-form-item v-if="createModel.db" path="db_user" :label="$gettext('Database User')">
            <n-input
              v-model:value="createModel.db_user"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Database User')"
            />
          </n-form-item>
        </n-col>
        <n-col :span="1"></n-col>
        <n-col :span="8">
          <n-form-item
            v-if="createModel.db"
            path="db_password"
            :label="$gettext('Database Password')"
          >
            <n-input
              v-model:value="createModel.db_password"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Database Password')"
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item path="path" :label="$gettext('Directory')">
        <n-input
          v-model:value="createModel.path"
          type="text"
          @keydown.enter.prevent
          :placeholder="
            $gettext(
              'Website root directory (if left empty, defaults to website directory/website name)'
            )
          "
        />
      </n-form-item>
      <n-form-item path="remark" :label="$gettext('Remark')">
        <n-input
          v-model:value="createModel.remark"
          type="textarea"
          @keydown.enter.prevent
          :placeholder="$gettext('Remark')"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleCreate">
      {{ $gettext('Create') }}
    </n-button>
  </n-modal>
  <n-modal
    v-model:show="editDefaultPageModal"
    preset="card"
    :title="$gettext('Modify Default Page')"
    style="width: 80vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="handleSaveDefaultPage"
  >
    <n-tabs type="line" animated>
      <n-tab-pane :name="$gettext('Default Page')" :tab="$gettext('Default Page')">
        <Editor
          v-model:value="editDefaultPageModel.index"
          language="html"
          theme="vs-dark"
          height="60vh"
          mt-8
          :options="{
            automaticLayout: true,
            formatOnType: true,
            formatOnPaste: true
          }"
        />
      </n-tab-pane>
      <n-tab-pane :name="$gettext('Stop Page')" :tab="$gettext('Stop Page')">
        <Editor
          v-model:value="editDefaultPageModel.stop"
          language="html"
          theme="vs-dark"
          height="60vh"
          mt-8
          :options="{
            automaticLayout: true,
            formatOnType: true,
            formatOnPaste: true
          }"
        />
      </n-tab-pane>
    </n-tabs>
  </n-modal>
  <bulk-create v-model:show="bulkCreateModal" />
</template>
