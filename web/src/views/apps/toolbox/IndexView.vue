<script setup lang="ts">
defineOptions({
  name: 'apps-toolbox-index'
})

import Editor from '@guolao/vue-monaco-editor'
import { DateTime } from 'luxon'
import { NButton } from 'naive-ui'

import toolbox from '@/api/apps/toolbox'

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
    window.$message.success('保存成功')
  })
}

const handleUpdateSwap = () => {
  useRequest(toolbox.updateSwap(swap.value)).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const handleUpdateHost = async () => {
  await Promise.all([
    useRequest(toolbox.updateHostname(hostname.value)),
    useRequest(toolbox.updateHosts(hosts.value))
  ]).then(() => {
    window.$message.success('保存成功')
  })
}

const handleUpdateRootPassword = () => {
  useRequest(toolbox.updateRootPassword(rootPassword.value)).onSuccess(() => {
    window.$message.success('保存成功')
  })
}

const handleUpdateTime = async () => {
  await Promise.all([
    useRequest(toolbox.updateTime(String(DateTime.fromMillis(time.value).toISO()))),
    useRequest(toolbox.updateTimezone(timezone.value))
  ]).then(() => {
    window.$message.success('保存成功')
  })
}

const handleSyncTime = () => {
  useRequest(toolbox.syncTime()).onSuccess(() => {
    window.$message.success('同步成功')
  })
}
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'dns'" class="ml-16" type="primary" @click="handleUpdateDNS">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        保存
      </n-button>
      <n-button v-if="currentTab == 'swap'" class="ml-16" type="primary" @click="handleUpdateSwap">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        保存
      </n-button>
      <n-button v-if="currentTab == 'host'" class="ml-16" type="primary" @click="handleUpdateHost">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        保存
      </n-button>
      <n-button v-if="currentTab == 'time'" class="ml-16" type="primary" @click="handleUpdateTime">
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        保存
      </n-button>
      <n-button
        v-if="currentTab == 'root-password'"
        class="ml-16"
        type="primary"
        @click="handleUpdateRootPassword"
      >
        <TheIcon :size="18" icon="material-symbols:save-outline" />
        修改
      </n-button>
    </template>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="dns" tab="DNS">
        <n-flex vertical>
          <n-alert type="warning"> DNS 修改后重启系统会还原默认。 </n-alert>
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
            总共 {{ swapTotal }}，已使用 {{ swapUsed }}，剩余 {{ swapFree }}
          </n-alert>
          <n-form>
            <n-form-item label="SWAP大小">
              <n-input-number v-model:value="swap" />
              MB
            </n-form-item>
          </n-form>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="host" tab="主机">
        <n-flex vertical>
          <n-form>
            <n-form-item label="主机名">
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
      <n-tab-pane name="time" tab="时间">
        <n-flex vertical>
          <n-alert type="info"> 手动修改时间后，仍有可能被系统自动同步时间覆盖。 </n-alert>
          <n-form>
            <n-form-item label="选择时区">
              <n-select v-model:value="timezone" placeholder="请选择时区" :options="timezones" />
            </n-form-item>
            <n-form-item label="修改时间">
              <n-date-picker v-model:value="time" type="datetime" clearable />
            </n-form-item>
            <n-form-item label="NTP 同步时间">
              <n-button type="info" @click="handleSyncTime">同步时间</n-button>
            </n-form-item>
          </n-form>
        </n-flex>
      </n-tab-pane>
      <n-tab-pane name="root-password" tab="Root 密码">
        <n-form>
          <n-form-item label="Root 密码">
            <n-input v-model:value="rootPassword" type="password" show-password-on="click" />
          </n-form-item>
        </n-form>
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>
