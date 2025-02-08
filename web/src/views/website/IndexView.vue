<script lang="ts" setup>
defineOptions({
  name: 'website-index'
})

import Editor from '@guolao/vue-monaco-editor'
import {
  NButton,
  NCheckbox,
  NDataTable,
  NFlex,
  NInput,
  NPopconfirm,
  NSpace,
  NSwitch,
  NTag
} from 'naive-ui'
import { useI18n } from 'vue-i18n'

import dashboard from '@/api/panel/dashboard'
import website from '@/api/panel/website'
import { useFileStore } from '@/store'
import { generateRandomString, isNullOrUndef, renderIcon } from '@/utils'

const fileStore = useFileStore()
const { t } = useI18n()
const router = useRouter()
const selectedRowKeys = ref<any>([])

const columns: any = [
  { type: 'selection', fixed: 'left' },
  {
    title: t('websiteIndex.columns.name'),
    key: 'name',
    width: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: t('websiteIndex.columns.status'),
    key: 'status',
    width: 150,
    align: 'center',
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
    title: t('websiteIndex.columns.path'),
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
    align: 'center',
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
    title: t('websiteIndex.columns.remark'),
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
    title: t('websiteIndex.columns.actions'),
    key: 'actions',
    width: 220,
    align: 'center',
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
            default: () => '修改',
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
                    h('strong', {}, { default: () => `确定删除网站 ${row.name} 吗？` }),
                    h(
                      NCheckbox,
                      {
                        checked: deleteModel.value.path,
                        onUpdateChecked: (v) => (deleteModel.value.path = v)
                      },
                      { default: () => '删除网站目录' }
                    ),
                    h(
                      NCheckbox,
                      {
                        checked: deleteModel.value.db,
                        onUpdateChecked: (v) => (deleteModel.value.db = v)
                      },
                      { default: () => '删除本地同名数据库' }
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

const createModal = ref(false)
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
        label: '不使用',
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
    window.$message.success('已' + (row.status ? '启动' : '停止'))
  })
}

const getDefaultPage = async () => {
  editDefaultPageModel.value = await website.defaultConfig()
}

const handleRemark = (row: any) => {
  useRequest(website.updateRemark(row.id, row.remark)).onSuccess(() => {
    window.$message.success('修改成功')
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
    window.$message.success('删除成功')
  })
}

const handleSaveDefaultPage = () => {
  useRequest(
    website.saveDefaultConfig(editDefaultPageModel.value.index, editDefaultPageModel.value.stop)
  ).onSuccess(() => {
    editDefaultPageModal.value = false
    window.$message.success('修改成功')
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
    window.$message.success('创建成功')
  })
}

const bulkDelete = async () => {
  if (selectedRowKeys.value.length === 0) {
    window.$message.info('请选择要删除的网站')
    return
  }

  const promises = selectedRowKeys.value.map((id: any) => {
    const site = data.value.find((item: any) => item.id === id)
    return useRequest(website.delete(id, true, false)).then(() => {
      window.$message.success('网站 ' + site?.name + ' 删除成功')
    })
  })

  await Promise.all(promises)
  await refresh()
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
})
</script>

<template>
  <common-page show-footer>
    <n-flex vertical size="large">
      <n-card rounded-10>
        <n-space>
          <n-button type="primary" @click="createModal = true">
            {{ $t('websiteIndex.create.trigger') }}
          </n-button>
          <n-popconfirm @positive-click="bulkDelete">
            <template #trigger>
              <n-button type="error"> 批量删除 </n-button>
            </template>
            这会删除网站目录但不会删除同名数据库，确定删除选中的网站吗？
          </n-popconfirm>
          <n-button type="warning" @click="editDefaultPageModal = true">
            {{ $t('websiteIndex.edit.trigger') }}
          </n-button>
        </n-space>
      </n-card>
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
    :title="$t('websiteIndex.create.title')"
    preset="card"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="createModal = false"
  >
    <n-form :model="createModel">
      <n-form-item path="name" :label="$t('websiteIndex.create.fields.name.label')">
        <n-input
          v-model:value="createModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('websiteIndex.create.fields.name.placeholder')"
        />
      </n-form-item>
      <n-row :gutter="[0, 24]">
        <n-col :span="11">
          <n-form-item :label="$t('websiteIndex.create.fields.domains.label')">
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
          <n-form-item :label="$t('websiteIndex.create.fields.port.label')">
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
          <n-form-item path="php" :label="$t('websiteIndex.create.fields.phpVersion.label')">
            <n-select
              v-model:value="createModel.php"
              :options="installedDbAndPhp.php"
              :placeholder="$t('websiteIndex.create.fields.phpVersion.placeholder')"
              @keydown.enter.prevent
            >
            </n-select>
          </n-form-item>
        </n-col>
        <n-col :span="2"></n-col>
        <n-col :span="11">
          <n-form-item path="db" :label="$t('websiteIndex.create.fields.db.label')">
            <n-select
              v-model:value="createModel.db_type"
              :options="installedDbAndPhp.db"
              :placeholder="$t('websiteIndex.create.fields.db.placeholder')"
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
          <n-form-item
            v-if="createModel.db"
            path="db_name"
            :label="$t('websiteIndex.create.fields.dbName.label')"
          >
            <n-input
              v-model:value="createModel.db_name"
              type="text"
              @keydown.enter.prevent
              :placeholder="$t('websiteIndex.create.fields.dbName.placeholder')"
            />
          </n-form-item>
        </n-col>
        <n-col :span="1"></n-col>
        <n-col :span="7">
          <n-form-item
            v-if="createModel.db"
            path="db_user"
            :label="$t('websiteIndex.create.fields.dbUser.label')"
          >
            <n-input
              v-model:value="createModel.db_user"
              type="text"
              @keydown.enter.prevent
              :placeholder="$t('websiteIndex.create.fields.dbUser.placeholder')"
            />
          </n-form-item>
        </n-col>
        <n-col :span="1"></n-col>
        <n-col :span="8">
          <n-form-item
            v-if="createModel.db"
            path="db_password"
            :label="$t('websiteIndex.create.fields.dbPassword.label')"
          >
            <n-input
              v-model:value="createModel.db_password"
              type="text"
              @keydown.enter.prevent
              :placeholder="$t('websiteIndex.create.fields.dbPassword.placeholder')"
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item path="path" :label="$t('websiteIndex.create.fields.path.label')">
        <n-input
          v-model:value="createModel.path"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('websiteIndex.create.fields.path.placeholder')"
        />
      </n-form-item>
      <n-form-item path="remark" :label="$t('websiteIndex.create.fields.remark.label')">
        <n-input
          v-model:value="createModel.remark"
          type="textarea"
          @keydown.enter.prevent
          :placeholder="$t('websiteIndex.create.fields.remark.placeholder')"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleCreate">
      {{ $t('websiteIndex.create.actions.submit') }}
    </n-button>
  </n-modal>
  <n-modal
    v-model:show="editDefaultPageModal"
    preset="card"
    title="修改默认页"
    style="width: 80vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="handleSaveDefaultPage"
  >
    <n-tabs type="line" animated>
      <n-tab-pane name="index" tab="默认页">
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
      <n-tab-pane name="stop" tab="停止页">
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
</template>
