<script setup lang="ts">
defineOptions({
  name: 'apps-fail2ban-index'
})

import { NButton, NDataTable, NInput, NPopconfirm, NSwitch } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import fail2ban from '@/api/apps/fail2ban'
import app from '@/api/panel/app'
import systemctl from '@/api/panel/systemctl'
import website from '@/api/panel/website'
import { renderIcon } from '@/utils'

const { $gettext } = useGettext()
const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)
const white = ref('')

const addJailModal = ref(false)
const addJailModel = ref({
  name: 'ssh',
  type: 'website',
  maxretry: 30,
  findtime: 300,
  bantime: 600,
  website_name: '',
  website_mode: 'cc',
  website_path: '/'
})

const jailModal = ref(false)
const jailCurrentlyBan = ref(0)
const jailTotalBan = ref(0)
const jailBanedList = ref<any[]>([])

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? $gettext('Running normally') : $gettext('Stopped')
})

const jailsColumns: any = [
  {
    title: $gettext('Name'),
    key: 'name',
    minWidth: 250,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Status'),
    key: 'enabled',
    minWidth: 60,
    align: 'center',
    render(row: any) {
      return h(NSwitch, {
        size: 'small',
        rubberBand: false,
        value: row.enabled,
        disabled: true
      })
    }
  },
  { title: $gettext('Max Retries'), key: 'max_retry', minWidth: 150, ellipsis: { tooltip: true } },
  { title: $gettext('Ban Time'), key: 'ban_time', minWidth: 150, ellipsis: { tooltip: true } },
  { title: $gettext('Find Time'), key: 'find_time', minWidth: 150, ellipsis: { tooltip: true } },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 280,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NButton,
          {
            size: 'small',
            type: 'warning',
            secondary: true,
            onClick: async () => {
              await getJailInfo(row.name)
              jailModal.value = true
            }
          },
          {
            default: () => $gettext('View'),
            icon: renderIcon('material-symbols:visibility', { size: 14 })
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDeleteJail(row.name)
          },
          {
            default: () => {
              return $gettext('Are you sure you want to delete rule %{ name }?', { name: row.name })
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

const banedIPColumns: any = [
  {
    title: 'IP',
    key: 'ip',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 100,
    align: 'center',
    hideInExcel: true,
    render(row: any) {
      return [
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleUnBan(row.name, row.ip)
          },
          {
            default: () => {
              return $gettext('Are you sure you want to unban %{ ip }?', { ip: row.ip })
            },
            trigger: () => {
              return h(
                NButton,
                {
                  size: 'small',
                  type: 'error'
                },
                {
                  default: () => $gettext('Unban'),
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

const websites = ref<any[]>([])

const getWhiteList = async () => {
  white.value = await fail2ban.whitelist()
}

const handleSaveWhiteList = () => {
  useRequest(fail2ban.setWhitelist(white.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const getWebsiteList = async (page: number, limit: number) => {
  const data = await website.list(page, limit)
  for (const item of data.items) {
    websites.value.push({
      label: item.name,
      value: item.name
    })
  }
  addJailModel.value.website_name = websites.value[0]?.value
}

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => fail2ban.jails(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const getStatus = async () => {
  status.value = await systemctl.status('fail2ban')
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled('fail2ban')
}

const handleStart = async () => {
  await systemctl.start('fail2ban')
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable('fail2ban')
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable('fail2ban')
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
}

const handleStop = async () => {
  await systemctl.stop('fail2ban')
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart('fail2ban')
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleReload = async () => {
  await systemctl.reload('fail2ban')
  window.$message.success($gettext('Reloaded successfully'))
  await getStatus()
}

const handleAddJail = () => {
  useRequest(fail2ban.add(addJailModel.value)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Added successfully'))
    addJailModal.value = false
  })
}

const handleDeleteJail = (name: string) => {
  useRequest(fail2ban.delete(name)).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Deleted successfully'))
  })
}

const getJailInfo = async (name: string) => {
  const data = await fail2ban.jail(name)
  jailCurrentlyBan.value = data.currently_ban
  jailTotalBan.value = data.total_ban
  jailBanedList.value = data.baned_list
}

const handleUnBan = (name: string, ip: string) => {
  useRequest(fail2ban.unban(name, ip)).onSuccess(() => {
    window.$message.success($gettext('Unbanned successfully'))
    getJailInfo(name)
  })
}

onMounted(() => {
  refresh()
  getStatus()
  getIsEnabled()
  getWhiteList()
  useRequest(app.isInstalled('nginx')).onSuccess(({ data }) => {
    if (data.installed) {
      getWebsiteList(1, 10000)
    }
  })
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button
        v-if="currentTab == 'status'"
        class="ml-16"
        type="primary"
        @click="handleSaveWhiteList"
      >
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save Whitelist') }}
      </n-button>
      <n-button
        v-if="currentTab == 'jails'"
        class="ml-16"
        type="primary"
        @click="addJailModal = true"
      >
        <TheIcon :size="18" icon="material-symbols:add" />
        {{ $gettext('Add Rule') }}
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
                  <TheIcon :size="24" icon="material-symbols:play-arrow-outline-rounded" />
                  {{ $gettext('Start') }}
                </n-button>
                <n-popconfirm @positive-click="handleStop">
                  <template #trigger>
                    <n-button type="error">
                      <TheIcon :size="24" icon="material-symbols:stop-outline-rounded" />
                      {{ $gettext('Stop') }}
                    </n-button>
                  </template>
                  {{ $gettext('Stopping Fail2ban will disable all rules. Are you sure you want to stop?') }}
                </n-popconfirm>
                <n-button type="warning" @click="handleRestart">
                  <TheIcon :size="18" icon="material-symbols:replay-rounded" />
                  {{ $gettext('Restart') }}
                </n-button>
                <n-button type="primary" @click="handleReload">
                  <TheIcon :size="20" icon="material-symbols:refresh-rounded" />
                  {{ $gettext('Reload') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
          <n-card :title="$gettext('IP Whitelist')">
            <n-input
              v-model:value="white"
              type="textarea"
              autosize
              :placeholder="$gettext('IP whitelist, separated by commas')"
            />
          </n-card>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="jails" :tab="$gettext('Rule Management')">
        <n-card :title="$gettext('Rule List')" :segmented="true">
          <n-data-table
            striped
            remote
            :scroll-x="1000"
            :loading="loading"
            :columns="jailsColumns"
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
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="fail2ban" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
  <n-modal v-model:show="addJailModal" :title="$gettext('Add Rule')">
    <n-card closable @close="() => (addJailModal = false)" :title="$gettext('Add Rule')" style="width: 60vw">
      <n-space vertical>
        <n-alert type="info">
          {{ $gettext('If an IP exceeds the maximum retries within the find time (seconds), it will be banned for the ban time (seconds)') }}
        </n-alert>
        <n-alert type="warning">
          {{ $gettext('Protected ports are automatically obtained. If you modify the port corresponding to a rule, please delete and re-add the rule, otherwise protection may not be effective') }}
        </n-alert>

        <n-form :model="addJailModel">
          <n-form-item :label="$gettext('Type')">
            <n-select
              v-model:value="addJailModel.type"
              :options="[
                { label: $gettext('Website'), value: 'website' },
                { label: $gettext('Service'), value: 'service' }
              ]"
            >
            </n-select>
          </n-form-item>
          <n-form-item v-if="addJailModel.type === 'website'" :label="$gettext('Select Website')">
            <n-select
              v-model:value="addJailModel.website_name"
              :options="websites"
              :placeholder="$gettext('Select Website')"
            />
          </n-form-item>
          <n-form-item v-if="addJailModel.type === 'website'" :label="$gettext('Protection Mode')">
            <n-select
              v-model:value="addJailModel.website_mode"
              :options="[
                { label: 'CC', value: 'cc' },
                { label: $gettext('Path'), value: 'path' }
              ]"
            >
            </n-select>
          </n-form-item>
          <n-form-item
            v-if="addJailModel.type === 'website' && addJailModel.website_mode === 'path'"
            :label="$gettext('Protection Path')"
          >
            <n-input v-model:value="addJailModel.website_path" :placeholder="$gettext('Protection Path')" />
          </n-form-item>
          <n-form-item v-if="addJailModel.type === 'service'" :label="$gettext('Service')">
            <n-select
              v-model:value="addJailModel.name"
              :options="[
                { label: 'SSH', value: 'ssh' },
                { label: 'MySQL', value: 'mysql' },
                { label: 'Pure-Ftpd', value: 'pure-ftpd' }
              ]"
            >
            </n-select>
          </n-form-item>
          <n-form-item path="maxretry" :label="$gettext('Max Retries')">
            <n-input-number v-model:value="addJailModel.maxretry" @keydown.enter.prevent :min="1" />
          </n-form-item>
          <n-form-item path="findtime" :label="$gettext('Find Time')">
            <n-input-number v-model:value="addJailModel.findtime" @keydown.enter.prevent :min="1" />
          </n-form-item>
          <n-form-item path="bantime" :label="$gettext('Ban Time')">
            <n-input-number v-model:value="addJailModel.bantime" @keydown.enter.prevent :min="1" />
          </n-form-item>
        </n-form>
        <n-button type="info" block @click="handleAddJail">{{ $gettext('Submit') }}</n-button>
      </n-space>
    </n-card>
  </n-modal>
  <n-modal v-model:show="jailModal" :title="$gettext('View Rule')">
    <n-card closable @close="() => (jailModal = false)" :title="$gettext('View Rule')" style="width: 60vw">
      <n-space vertical>
        <n-card :title="$gettext('Rule Information')" :segmented="true">
          <n-space vertical>
            <n-space>
              <n-text>{{ $gettext('Currently Banned') }}</n-text>
              <n-text>{{ jailCurrentlyBan }}</n-text>
            </n-space>
            <n-space>
              <n-text>{{ $gettext('Total Bans') }}</n-text>
              <n-text>{{ jailTotalBan }}</n-text>
            </n-space>
          </n-space>
        </n-card>
        <n-card :title="$gettext('Ban List')" :segmented="true">
          <n-data-table
            striped
            remote
            :scroll-x="300"
            :loading="false"
            :columns="banedIPColumns"
            :data="jailBanedList"
            :row-key="(row: any) => row.ip"
            :pagination="false"
          />
        </n-card>
      </n-space>
    </n-card>
  </n-modal>
</template>
