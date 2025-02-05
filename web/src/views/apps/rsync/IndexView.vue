<script setup lang="ts">
defineOptions({
  name: 'apps-rsync-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'

import rsync from '@/api/apps/rsync'
import systemctl from '@/api/panel/systemctl'
import { generateRandomString, renderIcon } from '@/utils'

const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)
const config = ref('')

const addModuleModal = ref(false)
const addModuleModel = ref({
  name: '',
  path: '/www',
  comment: '',
  auth_user: '',
  secret: generateRandomString(16),
  hosts_allow: '0.0.0.0/0'
})

const editModuleModal = ref(false)
const editModuleModel = ref({
  name: '',
  path: '',
  comment: '',
  auth_user: '',
  secret: '',
  hosts_allow: ''
})

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? '正常运行中' : '已停止运行'
})

const processColumns: any = [
  {
    title: '名称',
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '目录',
    key: 'path',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '用户',
    key: 'auth_user',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '主机',
    key: 'hosts_allow',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  { title: '备注', key: 'comment', resizable: true, ellipsis: { tooltip: true } },
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
            type: 'info',
            onClick: () => handleModelEdit(row)
          },
          {
            default: () => '配置',
            icon: renderIcon('material-symbols:settings-outline', { size: 14 })
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleModelDelete(row.name)
          },
          {
            default: () => {
              return '确定删除模块' + row.name + '吗？'
            },
            trigger: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'error',
                  style: 'margin-left: 15px'
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
  (page, pageSize) => rsync.modules(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const getStatus = async () => {
  status.value = await systemctl.status('rsyncd')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('rsyncd')
}

const getConfig = async () => {
  config.value = await rsync.config()
}

const handleSaveConfig = async () => {
  useRequest(rsync.saveConfig(config.value)).onSuccess(() => {
    refresh()
    window.$message.success('保存成功')
  })
}

const handleStart = async () => {
  await systemctl.start('rsyncd')
  window.$message.success('启动成功')
  await getStatus()
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('rsyncd')
    window.$message.success('开启自启动成功')
  } else {
    await systemctl.disable('rsyncd')
    window.$message.success('禁用自启动成功')
  }
  await getIsEnabled()
}

const handleStop = async () => {
  await systemctl.stop('rsyncd')
  window.$message.success('停止成功')
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('rsyncd')
  window.$message.success('重启成功')
  await getStatus()
}

const handleModelAdd = async () => {
  useRequest(rsync.addModule(addModuleModel.value)).onSuccess(() => {
    refresh()
    getConfig()
    addModuleModal.value = false
    addModuleModel.value = {
      name: '',
      path: '/www',
      comment: '',
      auth_user: '',
      secret: generateRandomString(16),
      hosts_allow: '0.0.0.0/0'
    }
    window.$message.success('添加成功')
  })
}

const handleModelDelete = async (name: string) => {
  useRequest(rsync.deleteModule(name)).onSuccess(() => {
    refresh()
    getConfig()
    window.$message.success('删除成功')
  })
}

const handleModelEdit = async (row: any) => {
  editModuleModal.value = true
  editModuleModel.value.name = row.name
  editModuleModel.value.path = row.path
  editModuleModel.value.comment = row.comment
  editModuleModel.value.auth_user = row.auth_user
  editModuleModel.value.secret = row.secret
  editModuleModel.value.hosts_allow = row.hosts_allow
}

const handleSaveModuleConfig = async () => {
  useRequest(rsync.updateModule(editModuleModel.value.name, editModuleModel.value)).onSuccess(
    () => {
      refresh()
      getConfig()
      window.$message.success('保存成功')
    }
  )
}

onMounted(() => {
  refresh()
  getStatus()
  getIsEnabled()
  getConfig()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button
        v-if="currentTab == 'config'"
        class="ml-16"
        type="primary"
        @click="handleSaveConfig"
      >
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        保存
      </n-button>
      <n-button
        v-if="currentTab == 'modules'"
        class="ml-16"
        type="primary"
        @click="addModuleModal = true"
      >
        <TheIcon :size="18" icon="material-symbols:add" />
        添加模块
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" tab="运行状态">
        <n-space vertical>
          <n-card title="运行状态" rounded-10>
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked> 自启动开 </template>
                <template #unchecked> 自启动关 </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="statusType">
                {{ statusStr }}
              </n-alert>
              <n-space>
                <n-button type="success" @click="handleStart">
                  <TheIcon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  启动
                </n-button>
                <n-popconfirm @positive-click="handleStop">
                  <template #trigger>
                    <n-button type="error">
                      <TheIcon :size="24" icon="material-symbols:stop-outline-rounded" />
                      停止
                    </n-button>
                  </template>
                  停止 Rsync 服务后，将无法使用 Rsync 功能，确定要停止吗？
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  重启
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="modules" tab="模块管理">
        <n-card title="模块列表" :segmented="true" rounded-10>
          <n-data-table
            striped
            remote
            :scroll-x="1000"
            :loading="loading"
            :columns="processColumns"
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
        </n-card>
      </n-tab-pane>
      <n-tab-pane name="config" tab="主配置">
        <n-space vertical>
          <n-alert type="warning">
            此处修改的是 Supervisor 主配置文件，如果您不了解各参数的含义，请不要随意修改！
          </n-alert>
          <Editor
            v-model:value="config"
            language="ini"
            theme="vs-dark"
            height="60vh"
            mt-8
            :options="{
              automaticLayout: true,
              formatOnType: true,
              formatOnPaste: true
            }"
          />
        </n-space>
      </n-tab-pane>
    </n-tabs>
  </common-page>
  <n-modal
    v-model:show="addModuleModal"
    preset="card"
    title="添加模块"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="addModuleModal = false"
  >
    <n-form :model="addModuleModel">
      <n-form-item path="name" label="名称">
        <n-input
          v-model:value="addModuleModel.name"
          type="text"
          @keydown.enter.prevent
          placeholder="名称禁止使用中文"
        />
      </n-form-item>
      <n-form-item path="path" label="目录">
        <n-input
          v-model:value="addModuleModel.path"
          type="text"
          @keydown.enter.prevent
          placeholder="请填写绝对路径"
        />
      </n-form-item>
      <n-form-item path="auth_user" label="用户">
        <n-input
          v-model:value="addModuleModel.auth_user"
          type="text"
          @keydown.enter.prevent
          placeholder="填写模块的用户名"
        />
      </n-form-item>
      <n-form-item path="secret" label="密码">
        <n-input
          v-model:value="addModuleModel.secret"
          type="text"
          @keydown.enter.prevent
          placeholder="填写模块的密码"
        />
      </n-form-item>
      <n-form-item path="hosts_allow" label="主机">
        <n-input
          v-model:value="addModuleModel.hosts_allow"
          type="text"
          @keydown.enter.prevent
          placeholder="填写允许访问的主机，多个主机用空格分隔"
        />
      </n-form-item>
      <n-form-item path="comment" label="备注">
        <n-input
          v-model:value="addModuleModel.comment"
          type="text"
          @keydown.enter.prevent
          placeholder="填写备注信息"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleModelAdd">提交</n-button>
  </n-modal>
  <n-modal
    v-model:show="editModuleModal"
    preset="card"
    title="模块配置"
    style="width: 80vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="handleSaveModuleConfig"
  >
    <n-form :model="editModuleModel">
      <n-form-item path="path" label="目录">
        <n-input
          v-model:value="editModuleModel.path"
          type="text"
          @keydown.enter.prevent
          placeholder="请填写绝对路径"
        />
      </n-form-item>
      <n-form-item path="auth_user" label="用户">
        <n-input
          v-model:value="editModuleModel.auth_user"
          type="text"
          @keydown.enter.prevent
          placeholder="填写模块的用户名"
        />
      </n-form-item>
      <n-form-item path="secret" label="密码">
        <n-input
          v-model:value="editModuleModel.secret"
          type="password"
          show-password-on="click"
          @keydown.enter.prevent
          placeholder="填写模块的密码"
        />
      </n-form-item>
      <n-form-item path="hosts_allow" label="主机">
        <n-input
          v-model:value="editModuleModel.hosts_allow"
          type="text"
          @keydown.enter.prevent
          placeholder="填写允许访问的主机，多个主机用空格分隔"
        />
      </n-form-item>
      <n-form-item path="comment" label="备注">
        <n-input
          v-model:value="editModuleModel.comment"
          type="text"
          @keydown.enter.prevent
          placeholder="填写备注信息"
        />
      </n-form-item>
    </n-form>
  </n-modal>
</template>
