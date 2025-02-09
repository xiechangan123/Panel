<script setup lang="ts">
import firewall from '@/api/panel/firewall'
import safe from '@/api/panel/safe'

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
    window.$message.success('设置成功')
  })
}

const handleSsh = () => {
  useRequest(safe.updateSsh(model.value.sshStatus, model.value.sshPort)).onSuccess(() => {
    window.$message.success('设置成功')
  })
}

const handlePingStatus = () => {
  useRequest(safe.updatePingStatus(model.value.pingStatus)).onSuccess(() => {
    window.$message.success('设置成功')
  })
}
</script>

<template>
  <n-form :model="model" label-placement="left" label-width="auto">
    <n-form-item path="firewall" label="系统防火墙">
      <n-switch v-model:value="model.firewallStatus" @update:value="handleFirewallStatus" />
    </n-form-item>
    <n-form-item path="ssh" label="SSH 开关">
      <n-switch v-model:value="model.sshStatus" @update:value="handleSsh" />
    </n-form-item>
    <n-form-item path="ping" label="允许 Ping">
      <n-switch v-model:value="model.pingStatus" @update:value="handlePingStatus" />
    </n-form-item>
    <n-form-item path="sshPort" label="SSH 端口">
      <n-input-number v-model:value="model.sshPort" @blur="handleSsh" />
    </n-form-item>
  </n-form>
</template>

<style scoped lang="scss"></style>
