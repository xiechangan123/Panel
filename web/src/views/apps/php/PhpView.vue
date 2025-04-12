<script setup lang="ts">
import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import php from '@/api/apps/php'
import systemctl from '@/api/panel/systemctl'
import { renderIcon } from '@/utils'

const { $gettext } = useGettext()
const props = defineProps({
  version: {
    type: Number,
    required: true
  }
})

const { version } = toRefs(props)

const currentTab = ref('status')
const status = ref(false)
const isEnabled = ref(false)

const { data: config } = useRequest(php.config(version.value), {
  initialData: ''
})
const { data: fpmConfig } = useRequest(php.fpmConfig(version.value), {
  initialData: ''
})
const { data: errorLog } = useRequest(php.errorLog(version.value), {
  initialData: ''
})
const { data: slowLog } = useRequest(php.slowLog(version.value), {
  initialData: ''
})
const { data: load } = useRequest(php.load(version.value), {
  initialData: []
})
const { data: extensions } = useRequest(php.extensions(version.value), {
  initialData: []
})

const statusType = computed(() => {
  return status.value ? 'success' : 'error'
})
const statusStr = computed(() => {
  return status.value ? $gettext('Running') : $gettext('Stopped')
})

const extensionColumns: any = [
  {
    title: $gettext('Extension Name'),
    key: 'name',
    minWidth: 250,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Description'),
    key: 'description',
    resizable: true,
    minWidth: 250,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 240,
    hideInExcel: true,
    render(row: any) {
      return [
        !row.installed
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => handleInstallExtension(row.slug)
              },
              {
                default: () => {
                  return $gettext('Are you sure you want to install %{ name }?', { name: row.name })
                },
                trigger: () => {
                  return h(
                    NButton,
                    {
                      size: 'small',
                      type: 'info'
                    },
                    {
                      default: () => $gettext('Install'),
                      icon: renderIcon('material-symbols:download-rounded', { size: 14 })
                    }
                  )
                }
              }
            )
          : null,
        row.installed
          ? h(
              NPopconfirm,
              {
                onPositiveClick: () => handleUninstallExtension(row.slug)
              },
              {
                default: () => {
                  return $gettext('Are you sure you want to uninstall %{ name }?', {
                    name: row.name
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
                      default: () => $gettext('Delete'),
                      icon: renderIcon('material-symbols:delete-outline', { size: 14 })
                    }
                  )
                }
              }
            )
          : null
      ]
    }
  }
]

const loadColumns: any = [
  {
    title: $gettext('Property'),
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Current Value'),
    key: 'value',
    minWidth: 200,
    ellipsis: { tooltip: true }
  }
]

const getStatus = async () => {
  status.value = await systemctl.status(`php-fpm-${version.value}`)
}

const getIsEnabled = async () => {
  isEnabled.value = await systemctl.isEnabled(`php-fpm-${version.value}`)
}

const handleSetCli = async () => {
  useRequest(php.setCli(version.value)).onSuccess(() => {
    window.$message.success($gettext('Set successfully'))
  })
}

const handleSaveConfig = async () => {
  useRequest(php.saveConfig(version.value, config.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleSaveFPMConfig = async () => {
  useRequest(php.saveFPMConfig(version.value, fpmConfig.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleClearErrorLog = async () => {
  useRequest(php.clearErrorLog(version.value)).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleClearSlowLog = async () => {
  useRequest(php.clearSlowLog(version.value)).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleIsEnabled = async () => {
  if (isEnabled.value) {
    await systemctl.enable(`php-fpm-${version.value}`)
    window.$message.success($gettext('Autostart enabled successfully'))
  } else {
    await systemctl.disable(`php-fpm-${version.value}`)
    window.$message.success($gettext('Autostart disabled successfully'))
  }
  await getIsEnabled()
}

const handleStart = async () => {
  await systemctl.start(`php-fpm-${version.value}`)
  window.$message.success($gettext('Started successfully'))
  await getStatus()
}

const handleStop = async () => {
  await systemctl.stop(`php-fpm-${version.value}`)
  window.$message.success($gettext('Stopped successfully'))
  await getStatus()
}

const handleRestart = async () => {
  await systemctl.restart(`php-fpm-${version.value}`)
  window.$message.success($gettext('Restarted successfully'))
  await getStatus()
}

const handleReload = async () => {
  await systemctl.reload(`php-fpm-${version.value}`)
  window.$message.success($gettext('Reloaded successfully'))
  await getStatus()
}

const handleInstallExtension = async (slug: string) => {
  useRequest(php.installExtension(version.value, slug)).onSuccess(() => {
    window.$message.success($gettext('Task submitted, please check progress in background tasks'))
  })
}

const handleUninstallExtension = async (name: string) => {
  useRequest(php.uninstallExtension(version.value, name)).onSuccess(() => {
    window.$message.success($gettext('Task submitted, please check progress in background tasks'))
  })
}

onMounted(() => {
  getStatus()
  getIsEnabled()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'status'" class="ml-16" type="info" @click="handleSetCli">
        {{ $gettext('Set as CLI Default Version') }}
      </n-button>
      <n-button
        v-if="currentTab == 'config'"
        class="ml-16"
        type="primary"
        @click="handleSaveConfig"
      >
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button
        v-if="currentTab == 'fpm-config'"
        class="ml-16"
        type="primary"
        @click="handleSaveFPMConfig"
      >
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button
        v-if="currentTab == 'error-log'"
        class="ml-16"
        type="primary"
        @click="handleClearErrorLog"
      >
        <TheIcon :size="18" icon="material-symbols:delete-outline" />
        {{ $gettext('Clear Error Log') }}
      </n-button>
      <n-button
        v-if="currentTab == 'slow-log'"
        class="ml-16"
        type="primary"
        @click="handleClearSlowLog"
      >
        <TheIcon :size="18" icon="material-symbols:delete-outline" />
        {{ $gettext('Clear Slow Log') }}
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
                  {{
                    $gettext(
                      'Stopping PHP %{ version } will cause websites using PHP %{ version } to become inaccessible. Are you sure you want to stop?',
                      { version: version }
                    )
                  }}
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
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="extensions" :tab="$gettext('Extension Management')">
        <n-card :title="$gettext('Extension List')" :segmented="true">
          <n-data-table
            striped
            remote
            :scroll-x="1000"
            :loading="false"
            :columns="extensionColumns"
            :data="extensions"
            :row-key="(row: any) => row.slug"
          />
        </n-card>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Main Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the PHP %{ version } main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!',
                { version: version }
              )
            }}
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
      <n-tab-pane name="fpm-config" :tab="$gettext('FPM Configuration')">
        <n-space vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the PHP %{ version } FPM configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!',
                { version: version }
              )
            }}
          </n-alert>
          <Editor
            v-model:value="fpmConfig"
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
      <n-tab-pane name="load" :tab="$gettext('Load Status')">
        <n-data-table
          striped
          remote
          :scroll-x="400"
          :loading="false"
          :columns="loadColumns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log :service="'php-fpm-' + version" />
      </n-tab-pane>
      <n-tab-pane name="error-log" :tab="$gettext('Error Logs')">
        <realtime-log :path="errorLog" />
      </n-tab-pane>
      <n-tab-pane name="slow-log" :tab="$gettext('Slow Logs')">
        <realtime-log :path="slowLog" />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
