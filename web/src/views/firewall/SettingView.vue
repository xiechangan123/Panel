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
</script>

<template>
  <n-form :model="model" label-placement="left" label-width="auto">
    <n-form-item path="firewall" :label="$gettext('System Firewall')">
      <n-switch v-model:value="model.firewallStatus" @update:value="handleFirewallStatus" />
    </n-form-item>
    <n-form-item path="ping" :label="$gettext('Allow Ping')">
      <n-switch v-model:value="model.pingStatus" @update:value="handlePingStatus" />
    </n-form-item>
  </n-form>
</template>

<style scoped lang="scss"></style>
