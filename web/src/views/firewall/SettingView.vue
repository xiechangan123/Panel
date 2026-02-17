<script setup lang="ts">
import firewall from '@/api/panel/firewall'
import safe from '@/api/panel/safe'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const model = ref({
  firewallStatus: false,
  pingStatus: false
})

useRequest(firewall.status).onSuccess(({ data }) => {
  model.value.firewallStatus = data
})
useRequest(safe.pingStatus).onSuccess(({ data }) => {
  model.value.pingStatus = data
})

const handleFirewallStatus = () => {
  useRequest(firewall.updateStatus(model.value.firewallStatus)).onSuccess(() => {
    window.$message.success($gettext('Settings saved successfully'))
  })
}

const handlePingStatus = () => {
  useRequest(safe.updatePingStatus(model.value.pingStatus)).onSuccess(() => {
    window.$message.success($gettext('Settings saved successfully'))
  })
}

// 扫描感知设置
const scanEnabled = ref(false)
const saveDays = ref(30)
const selectedInterfaces = ref<string[]>([])
const interfaceOptions = ref<any[]>([])
const scanSettingLoading = ref(false)

useRequest(firewall.scanSetting()).onSuccess(({ data }) => {
  scanEnabled.value = data.enabled
  saveDays.value = data.days || 30
  selectedInterfaces.value = data.interfaces || []
})

useRequest(firewall.scanInterfaces()).onSuccess(({ data }) => {
  interfaceOptions.value = (data || []).map((iface: any) => ({
    label: `${iface.name} (${iface.ips?.join(', ') || iface.status})`,
    value: iface.name
  }))
})

const handleSaveScanSetting = () => {
  scanSettingLoading.value = true
  useRequest(
    firewall.updateScanSetting({
      enabled: scanEnabled.value,
      days: saveDays.value,
      interfaces: selectedInterfaces.value
    })
  )
    .onSuccess(() => {
      window.$message.success($gettext('Settings saved successfully'))
    })
    .onComplete(() => {
      scanSettingLoading.value = false
    })
}
</script>

<template>
  <n-form :model="model" label-placement="left" label-width="auto">
    <n-form-item path="firewall" :label="$gettext('System Firewall')">
      <n-switch v-model:value="model.firewallStatus" @update:value="handleFirewallStatus" />
    </n-form-item>
    <n-form-item path="ping" :label="$gettext('Allow Ping')">
      <n-switch v-model:value="model.pingStatus" @update:value="handlePingStatus" />
    </n-form-item>
    <n-divider />
    <n-form-item :label="$gettext('Scan Awareness')">
      <n-switch v-model:value="scanEnabled" />
    </n-form-item>
    <n-form-item :label="$gettext('Retention Days')">
      <n-input-number v-model:value="saveDays" :min="1" :max="365" w-full />
    </n-form-item>
    <n-form-item :label="$gettext('Network Interfaces')">
      <n-select
        v-model:value="selectedInterfaces"
        :options="interfaceOptions"
        multiple
        clearable
        :placeholder="$gettext('Auto detect')"
      />
    </n-form-item>
    <n-form-item>
      <n-button type="primary" :loading="scanSettingLoading" @click="handleSaveScanSetting">
        {{ $gettext('Save') }}
      </n-button>
    </n-form-item>
  </n-form>
</template>

<style scoped lang="scss"></style>
