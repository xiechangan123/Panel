<script setup lang="ts">
defineOptions({
  name: 'apps-openlitespeed-index',
})

import { NSwitch, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import openlitespeed from '@/api/apps/openlitespeed'
import ServiceStatus from '@/components/common/ServiceStatus.vue'

const { $gettext } = useGettext()
const currentTab = ref('status')
const saveConfigLoading = ref(false)
const clearErrorLogLoading = ref(false)

const { data: config } = useRequest(openlitespeed.config, {
  initialData: '',
})
const { data: errorLog } = useRequest(openlitespeed.errorLog, {
  initialData: '',
})
const { data: load } = useRequest(openlitespeed.load, {
  initialData: [],
})
const { data: phpList, send: fetchPHP } = useRequest(openlitespeed.php, {
  initialData: [],
})

const columns: any = [
  {
    title: $gettext('Property'),
    key: 'name',
    minWidth: 200,
    resizable: true,
    ellipsis: { tooltip: true },
  },
  {
    title: $gettext('Current Value'),
    key: 'value',
    minWidth: 200,
    ellipsis: { tooltip: true },
  },
]

const phpColumns: any = [
  {
    title: $gettext('PHP Version'),
    key: 'version',
    minWidth: 120,
    render(row: any) {
      return 'PHP ' + String(row.version).replace(/(\d)(\d)/, '$1.$2')
    },
  },
  {
    title: 'LSPHP',
    key: 'lsphp',
    minWidth: 120,
    render(row: any) {
      return h(
        NTag,
        { type: row.lsphp ? 'success' : 'warning', size: 'small' },
        { default: () => (row.lsphp ? $gettext('Available') : $gettext('Not Available')) },
      )
    },
  },
  {
    title: $gettext('Protocol'),
    key: 'lsapi',
    minWidth: 200,
    render(row: any) {
      return h(
        NSwitch,
        {
          value: row.lsapi,
          disabled: !row.lsphp,
          checkedValue: true,
          uncheckedValue: false,
          onUpdateValue: (value: boolean) => handleSetPHP(row.version, value),
        },
        {
          checked: () => 'LSAPI',
          unchecked: () => 'FastCGI',
        },
      )
    },
  },
]

const handleSetPHP = (version: number, lsapi: boolean) => {
  useRequest(openlitespeed.setPHP(version, lsapi)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
    fetchPHP()
  })
}

const handleSaveConfig = () => {
  saveConfigLoading.value = true
  useRequest(openlitespeed.saveConfig(config.value))
    .onSuccess(() => {
      window.$message.success($gettext('Saved successfully'))
    })
    .onComplete(() => {
      saveConfigLoading.value = false
    })
}

const errorLogRef = ref<{ clear: () => void } | null>(null)

const handleClearErrorLog = () => {
  clearErrorLogLoading.value = true
  useRequest(openlitespeed.clearErrorLog())
    .onSuccess(() => {
      errorLogRef.value?.clear()
      window.$message.success($gettext('Cleared successfully'))
    })
    .onComplete(() => {
      clearErrorLogLoading.value = false
    })
}
</script>

<template>
  <PageContainer :show-footer="true">
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="status" :tab="$gettext('Running Status')">
        <service-status service="openlitespeed" show-reload />
      </n-tab-pane>
      <n-tab-pane name="config" :tab="$gettext('Modify Configuration')">
        <n-flex vertical>
          <n-alert type="warning">
            {{
              $gettext(
                'This modifies the %{name} main configuration file. If you do not understand the meaning of each parameter, please do not modify it randomly!',
                { name: 'OpenLiteSpeed' },
              )
            }}
          </n-alert>
          <common-editor v-model:value="config" lang="plaintext" height="60vh" />
          <n-flex>
            <n-button
              type="primary"
              :loading="saveConfigLoading"
              :disabled="saveConfigLoading"
              @click="handleSaveConfig"
            >
              {{ $gettext('Save') }}
            </n-button>
          </n-flex>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="php" :tab="$gettext('PHP Protocol')">
        <n-flex vertical>
          <n-alert type="info">
            {{
              $gettext(
                'LSAPI is the native protocol of OpenLiteSpeed and performs better than FastCGI. It requires the LSPHP binary shipped with the panel PHP; if it is not available, reinstall the PHP version first.',
              )
            }}
          </n-alert>
          <n-data-table striped remote :scroll-x="400" :columns="phpColumns" :data="phpList" />
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="load" :tab="$gettext('Load Status')">
        <n-data-table
          striped
          remote
          :scroll-x="400"
          :loading="false"
          :columns="columns"
          :data="load"
        />
      </n-tab-pane>
      <n-tab-pane name="run-log" :tab="$gettext('Runtime Logs')">
        <realtime-log service="openlitespeed" />
      </n-tab-pane>
      <n-tab-pane name="error-log" :tab="$gettext('Error Logs')">
        <n-flex vertical>
          <n-flex>
            <n-button
              type="primary"
              :loading="clearErrorLogLoading"
              :disabled="clearErrorLogLoading"
              @click="handleClearErrorLog"
            >
              {{ $gettext('Clear Log') }}
            </n-button>
          </n-flex>
          <realtime-log ref="errorLogRef" :path="errorLog" />
        </n-flex>
      </n-tab-pane>
    </n-tabs>
  </PageContainer>
</template>
