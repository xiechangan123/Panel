<script setup lang="ts">
defineOptions({
  name: 'apps-toolbox-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { DateTime } from 'luxon'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import toolbox from '@/api/apps/toolbox'

const { $gettext } = useGettext()
const currentTab = ref('dns')
const dns1 = ref('')
const dns2 = ref('')
const swap = ref(0)
const swapFree = ref('')
const swapUsed = ref('')
const swapTotal = ref('')
const hostname = ref('')
const hosts = ref('')
const timezone = ref('')
const timezones = ref<any[]>([])
const time = ref(DateTime.now().toMillis())
const rootPassword = ref('')

useRequest(toolbox.dns()).onSuccess(({ data }) => {
  dns1.value = data[0]
  dns2.value = data[1]
})
useRequest(toolbox.swap()).onSuccess(({ data }) => {
  swap.value = data.size
  swapFree.value = data.free
  swapUsed.value = data.used
  swapTotal.value = data.total
})
useRequest(toolbox.hostname()).onSuccess(({ data }) => {
  hostname.value = data
})
useRequest(toolbox.hosts()).onSuccess(({ data }) => {
  hosts.value = data
})
useRequest(toolbox.timezone()).onSuccess(({ data }) => {
  timezone.value = data.timezone
  timezones.value = data.timezones
})

const handleUpdateDNS = () => {
  useRequest(toolbox.updateDns(dns1.value, dns2.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleUpdateSwap = () => {
  useRequest(toolbox.updateSwap(swap.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleUpdateHost = async () => {
  await Promise.all([
    useRequest(toolbox.updateHostname(hostname.value)),
    useRequest(toolbox.updateHosts(hosts.value))
  ]).then(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleUpdateRootPassword = () => {
  useRequest(toolbox.updateRootPassword(rootPassword.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleUpdateTime = async () => {
  await Promise.all([
    useRequest(toolbox.updateTime(String(DateTime.fromMillis(time.value).toISO()))),
    useRequest(toolbox.updateTimezone(timezone.value))
  ]).then(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

const handleSyncTime = () => {
  useRequest(toolbox.syncTime()).onSuccess(() => {
    window.$message.success($gettext('Synchronized successfully'))
  })
}
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'dns'" class="ml-16" type="primary" @click="handleUpdateDNS">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button v-if="currentTab == 'swap'" class="ml-16" type="primary" @click="handleUpdateSwap">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button v-if="currentTab == 'host'" class="ml-16" type="primary" @click="handleUpdateHost">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button v-if="currentTab == 'time'" class="ml-16" type="primary" @click="handleUpdateTime">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Save') }}
      </n-button>
      <n-button
        v-if="currentTab == 'root-password'"
        class="ml-16"
        type="primary"
        @click="handleUpdateRootPassword"
      >
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        {{ $gettext('Modify') }}
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="dns" tab="DNS">
        <n-flex vertical>
          <n-alert type="warning">
            {{ $gettext('DNS modifications will revert to default after system restart.') }}
          </n-alert>
          <n-form>
            <n-form-item label="DNS1">
              <n-input v-model:value="dns1" />
            </n-form-item>
            <n-form-item label="DNS2">
              <n-input v-model:value="dns2" />
            </n-form-item>
          </n-form>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="swap" tab="SWAP">
        <n-flex vertical>
          <n-alert type="info">
            {{
              $gettext('Total %{ total }, used %{ used }, free %{ free }', {
                total: swapTotal,
                used: swapUsed,
                free: swapFree
              })
            }}
          </n-alert>
          <n-form>
            <n-form-item :label="$gettext('SWAP Size')">
              <n-input-number v-model:value="swap" />
              MB
            </n-form-item>
          </n-form>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="host" :tab="$gettext('Host')">
        <n-flex vertical>
          <n-form>
            <n-form-item :label="$gettext('Hostname')">
              <n-input v-model:value="hostname" />
            </n-form-item>
          </n-form>
          <Editor
            v-model:value="hosts"
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
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="time" :tab="$gettext('Time')">
        <n-flex vertical>
          <n-alert type="info">
            {{
              $gettext(
                'After manually changing the time, it may still be overwritten by system automatic time synchronization.'
              )
            }}
          </n-alert>
          <n-form>
            <n-form-item :label="$gettext('Select Timezone')">
              <n-select
                v-model:value="timezone"
                :placeholder="$gettext('Please select a timezone')"
                :options="timezones"
              />
            </n-form-item>
            <n-form-item :label="$gettext('Modify Time')">
              <n-date-picker v-model:value="time" type="datetime" clearable />
            </n-form-item>
            <n-form-item :label="$gettext('NTP Time Synchronization')">
              <n-button type="info" @click="handleSyncTime">{{
                $gettext('Synchronize Time')
              }}</n-button>
            </n-form-item>
          </n-form>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="root-password" :tab="$gettext('Root Password')">
        <n-form>
          <n-form-item :label="$gettext('Root Password')">
            <n-input v-model:value="rootPassword" type="password" show-password-on="click" />
          </n-form-item>
        </n-form>
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
