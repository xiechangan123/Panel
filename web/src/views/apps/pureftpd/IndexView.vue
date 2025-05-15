<script setup lang="ts">
defineOptions({
  name: 'apps-pureftpd-index'
})

import { NButton, NDataTable, NInput, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import pureftpd from '@/api/apps/pureftpd'
import systemctl from '@/api/panel/systemctl'
import { generateRandomString, renderIcon } from '@/utils'

const { $gettext } = useGettext()
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
  return status.value ? $gettext('Running') : $gettext('Stopped')
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
    title: $gettext('Username'),
    key: 'username',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Path'),
    key: 'path',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 240,
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
            default: () => $gettext('Change Password'),
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
              return $gettext('Are you sure you want to delete user %{ username }?', {
                username: row.username
              })
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
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleStart = async () => {
  await systemctl.start('pure-ftpd')
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('pure-ftpd')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('pure-ftpd')
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
}

const handleStop = async () => {
  await systemctl.stop('pure-ftpd')
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('pure-ftpd')
  window.$message.success($gettext('Restarted successfully'))
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
    window.$message.success($gettext('Added successfully'))
  })
}

const handleChangePassword = async () => {
  useRequest(
    pureftpd.changePassword(changePasswordModel.value.username, changePasswordModel.value.password)
  ).onSuccess(() => {
    refresh()
    changePasswordModal.value = false
    window.$message.success($gettext('Modified successfully'))
  })
}

const handleDeleteUser = async (username: string) => {
  useRequest(pureftpd.delete(username)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Deleted successfully'))
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
        <the-icon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button
        v-if="currentTab == 'users'"
        class="ml-16"
        type="primary"
        @click="addUserModal = true"
      >
        <the-icon :size="18" icon="material-symbols:add" />
        {{ $gettext('Add User') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <n-space vertical>
          <n-card :title="$gettext('Running Status')">
            <template #header-extra>
              <n-switch v-model:value="isEnabled" @update:value="handleIsEnabled">
                <template #checked> {{ $gettext('Autostart On') }} </template>
                <template #unchecked> {{ $gettext('Autostart Off') }} </template>
              </n-switch>
            </template>
            <n-space vertical>
              <n-alert :type="statusType">
                {{ statusStr }}
              </n-alert>
              <n-space>
                <n-button type="success" @click="handleStart">
                  <the-icon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  {{ $gettext('Start') }}
                </n-button>
                <n-popconfirm @positive-click="handleStop">
                  <template #trigger>
                    <n-button type="error">
                      <the-icon :size="24" icon="material-symbols:stop-outline-rounded" />
                      {{ $gettext('Stop') }}
                    </n-button>
                  </template>
                  {{
                    $gettext(
                      'Stopping Pure-Ftpd will cause FTP service to be unavailable. Are you sure you want to stop it?'
                    )
                  }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <the-icon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
          <n-card :title="$gettext('Port Settings')">
            <n-input-number v-model:value="port" :min="1" :max="65535" />
            {{ $gettext('Modify Pure-Ftpd listening port') }}
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="users" :tab="$gettext('User Management')">
        <n-card :title="$gettext('User List')" :segmented="true">
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
      <n-tab-pane name="run-log" :tab="$gettext('Run Log')">
        <realtime-log service="pure-ftpd" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
  <n-modal v-model:show="addUserModal" :title="$gettext('Create User')">
    <n-card
      closable
      @close="() => (addUserModal = false)"
      :title="$gettext('Create User')"
      style="width: 60vw"
    >
      <n-form :model="addUserModel">
        <n-form-item path="username" :label="$gettext('Username')">
          <n-input
            v-model:value="addUserModel.username"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter username')"
          />
        </n-form-item>
        <n-form-item path="password" :label="$gettext('Password')">
          <n-input
            v-model:value="addUserModel.password"
            type="password"
            show-password-on="click"
            @keydown.enter.prevent
            :placeholder="
              $gettext('It is recommended to use the generator to generate a random password')
            "
          />
        </n-form-item>
        <n-form-item path="path" :label="$gettext('Directory')">
          <n-input
            v-model:value="addUserModel.path"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter the directory authorized to the user')"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleAddUser">{{ $gettext('Submit') }}</n-button>
    </n-card>
  </n-modal>
  <n-modal v-model:show="changePasswordModal">
    <n-card
      closable
      @close="() => (changePasswordModal = false)"
      :title="$gettext('Change Password')"
      style="width: 60vw"
    >
      <n-form :model="changePasswordModel">
        <n-form-item path="password" :label="$gettext('Password')">
          <n-input
            v-model:value="changePasswordModel.password"
            type="text"
            @keydown.enter.prevent
            :placeholder="
              $gettext('It is recommended to use the generator to generate a random password')
            "
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleChangePassword">{{ $gettext('Submit') }}</n-button>
    </n-card>
  </n-modal>
</template>
