<script setup lang="ts">
import Editor from '@guolao/vue-monaco-editor'
import { NButton, NDataTable, NPopconfirm } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import php from '@/api/apps/php'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const props = defineProps({
  version: {
    type: Number,
    required: true
  }
})

const { version } = toRefs(props)

const currentTab = ref('status')

const { data: config } = useRequest(php.config(version.value), {
  initialData: ''
})
const { data: fpmConfig } = useRequest(php.fpmConfig(version.value), {
  initialData: ''
})
const { data: log } = useRequest(php.log(version.value), {
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
                      default: () => $gettext('Install')
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
                      default: () => $gettext('Delete')
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

const handleClearLog = async () => {
  useRequest(php.clearLog(version.value)).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

const handleClearSlowLog = async () => {
  useRequest(php.clearSlowLog(version.value)).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
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
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <n-flex vertical>
          <service-status :service="`php-fpm-${version}`" show-reload />
          <n-button type="info" @click="handleSetCli">
            {{ $gettext('Set as CLI Default Version') }}
          </n-button>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="extensions" :tab="$gettext('Extension Management')">
        <n-flex vertical>
          <n-data-table
            striped
            remote
            :scroll-x="1000"
            :loading="false"
            :columns="extensionColumns"
            :data="extensions"
            :row-key="(row: any) => row.slug"
          />
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Main Configuration')">
        <n-flex vertical>
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
              smoothScrolling: true
            }"
          />
          <n-flex>
            <n-button type="primary" @click="handleSaveConfig">
              {{ $gettext('Save') }}
            </n-button>
          </n-flex>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="fpm-config" :tab="$gettext('FPM Configuration')">
        <n-flex vertical>
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
              smoothScrolling: true
            }"
          />
          <n-flex>
            <n-button type="primary" @click="handleSaveFPMConfig">
              {{ $gettext('Save') }}
            </n-button>
          </n-flex>
        </n-flex>
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
      <n-tab-pane name="log" :tab="$gettext('Error Logs')">
        <n-flex vertical>
          <n-flex>
            <n-button type="primary" @click="handleClearLog">
              {{ $gettext('Clear Log') }}
            </n-button>
          </n-flex>
          <realtime-log :path="log" />
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="slow-log" :tab="$gettext('Slow Logs')">
        <n-flex vertical>
          <n-flex>
            <n-button type="primary" @click="handleClearSlowLog">
              {{ $gettext('Clear Slow Log') }}
            </n-button>
          </n-flex>
          <realtime-log :path="slowLog" />
        </n-flex>
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
