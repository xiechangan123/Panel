<script setup lang="ts">
import firewall from '@/api/panel/firewall'
import safe from '@/api/panel/safe'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const model = ref({
  firewallStatus: false,
  sshStatus: false,
  pingStatus: false,
  sshPort: 22
})

useRequest(firewall.status).onSuccess(({ data }) => {
  model.value.firewallStatus = data
})
useRequest(safe.ssh).onSuccess(({ data }) => {
  model.value.sshStatus = data.status
  model.value.sshPort = data.port
})
useRequest(safe.pingStatus).onSuccess(({ data }) => {
  model.value.pingStatus = data
})

const handleFirewallStatus = () => {
  useRequest(firewall.updateStatus(model.value.firewallStatus)).onSuccess(() => {
    window.$message.success($gettext('Settings saved successfully'))
  })
}

const handleSsh = () => {
  useRequest(safe.updateSsh(model.value.sshStatus, model.value.sshPort)).onSuccess(() => {
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
    <n-form-item path="ssh" :label="$gettext('SSH Switch')">
      <n-switch v-model:value="model.sshStatus" @update:value="handleSsh" />
    </n-form-item>
    <n-form-item path="ping" :label="$gettext('Allow Ping')">
      <n-switch v-model:value="model.pingStatus" @update:value="handlePingStatus" />
    </n-form-item>
    <n-form-item path="sshPort" :label="$gettext('SSH Port')">
      <n-input-number v-model:value="model.sshPort" @blur="handleSsh" />
    </n-form-item>
  </n-form>
</template>

<style scoped lang="scss"></style>
