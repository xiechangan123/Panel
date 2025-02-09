<script setup lang="ts">
defineOptions({
  name: 'apps-pureftpd-index'
})

import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'

import pureftpd from '@/api/apps/pureftpd'
import systemctl from '@/api/panel/systemctl'
import { generateRandomString, renderIcon } from '@/utils'

const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)
const port = ref(0)
const addUserModal = ref(false)
const changePasswordModal = ref(false)

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? '正常运行中' : '已停止运行'
})

const addUserModel = ref({
  username: '',
  password: generateRandomString(16),
  path: ''
})

const changePasswordModel = ref({
  username: '',
  password: generateRandomString(16)
})

const userColumns: any = [
  {
    title: '用户名',
    key: 'username',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '路径',
    key: 'path',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NButton,
          {
            size: 'small',
            type: 'primary',
            secondary: true,
            onClick: () => {
              changePasswordModel.value.username = row.username
              changePasswordModel.value.password = generateRandomString(16)
              changePasswordModal.value = true
            }
          },
          {
            default: () => '改密',
            icon: renderIcon('material-symbols:key-outline', { size: 14 })
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDeleteUser(row.username)
          },
          {
            default: () => {
              return '确定删除用户' + row.username + '吗？'
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
  (page, pageSize) => pureftpd.list(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const getStatus = async () => {
  status.value = await systemctl.status('pure-ftpd')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('pure-ftpd')
}

const getPort = async () => {
  port.value = await pureftpd.port()
}

const handleSavePort = async () => {
  useRequest(pureftpd.updatePort(port.value)).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const handleStart = async () => {
  await systemctl.start('pure-ftpd')
  window.$message.success('启动成功')
  await getStatus()
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('pure-ftpd')
    window.$message.success('开启自启动成功')
  } else {
    await systemctl.disable('pure-ftpd')
    window.$message.success('禁用自启动成功')
  }
  await getIsEnabled()
}

const handleStop = async () => {
  await systemctl.stop('pure-ftpd')
  window.$message.success('停止成功')
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('pure-ftpd')
  window.$message.success('重启成功')
  await getStatus()
}

const handleAddUser = async () => {
  useRequest(
    pureftpd.add(addUserModel.value.username, addUserModel.value.password, addUserModel.value.path)
  ).onSuccess(() => {
    refresh()
    addUserModal.value = false
    addUserModel.value.username = ''
    addUserModel.value.password = generateRandomString(16)
    addUserModel.value.path = ''
    window.$message.success('添加成功')
  })
}

const handleChangePassword = async () => {
  useRequest(
    pureftpd.changePassword(changePasswordModel.value.username, changePasswordModel.value.password)
  ).onSuccess(() => {
    refresh()
    changePasswordModal.value = false
    window.$message.success('修改成功')
  })
}

const handleDeleteUser = async (username: string) => {
  useRequest(pureftpd.delete(username)).onSuccess(() => {
    refresh()
    window.$message.success('删除成功')
  })
}

onMounted(() => {
  refresh()
  getStatus()
  getIsEnabled()
  getPort()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'status'" class="ml-16" type="primary" @click="handleSavePort">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        保存
      </n-button>
      <n-button
        v-if="currentTab == 'users'"
        class="ml-16"
        type="primary"
        @click="addUserModal = true"
      >
        <TheIcon :size="18" icon="material-symbols:add" />
        添加用户
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" tab="运行状态">
        <n-space vertical>
          <n-card title="运行状态">
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
                  停止 Pure-Ftpd 会导致无法使用 FTP 服务，确定要停止吗？
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  重启
                </n-button>
              </n-space>
            </n-space>
          </n-card>
          <n-card title="端口设置">
            <n-input-number v-model:value="port" :min="1" :max="65535" />
            修改 Pure-Ftpd 监听端口
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="users" tab="用户管理">
        <n-card title="用户列表" :segmented="true">
          <n-data-table
            striped
            remote
            :scroll-x="1000"
            :loading="loading"
            :columns="userColumns"
            :data="data"
            :row-key="(row: any) => row.username"
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
    </n-tabs>
  </common-page>
  <n-modal v-model:show="addUserModal" title="创建用户">
    <n-card closable @close="() => (addUserModal = false)" title="创建用户" style="width: 60vw">
      <n-form :model="addUserModel">
        <n-form-item path="username" label="用户名">
          <n-input
            v-model:value="addUserModel.username"
            type="text"
            @keydown.enter.prevent
            placeholder="输入用户名"
          />
        </n-form-item>
        <n-form-item path="password" label="密码">
          <n-input
            v-model:value="addUserModel.password"
            type="password"
            show-password-on="click"
            @keydown.enter.prevent
            placeholder="建议使用生成器生成随机密码"
          />
        </n-form-item>
        <n-form-item path="path" label="目录">
          <n-input
            v-model:value="addUserModel.path"
            type="text"
            @keydown.enter.prevent
            placeholder="输入授权给该用户的目录"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleAddUser">提交</n-button>
    </n-card>
  </n-modal>
  <n-modal v-model:show="changePasswordModal">
    <n-card
      closable
      @close="() => (changePasswordModal = false)"
      title="修改密码"
      style="width: 60vw"
    >
      <n-form :model="changePasswordModel">
        <n-form-item path="password" label="密码">
          <n-input
            v-model:value="changePasswordModel.password"
            type="text"
            @keydown.enter.prevent
            placeholder="建议使用生成器生成随机密码"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleChangePassword">提交</n-button>
    </n-card>
  </n-modal>
</template>
