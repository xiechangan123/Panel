<script setup lang="ts">
import firewall from '@/api/panel/firewall'
import safe from '@/api/panel/safe'

const model = ref({
  firewallStatus: false,
  sshStatus: false,
  pingStatus: false,
  sshPort: 22
})

const fetchSetting = async () => {
  firewall.status().then((res) => {
    model.value.firewallStatus = res.data
  })

  const ssh = await safe.ssh()
  model.value.sshStatus = ssh.status
  model.value.sshPort = ssh.port

  model.value.pingStatus = await safe.pingStatus()
}

const handleFirewallStatus = () => {
  firewall.updateStatus(model.value.firewallStatus).then(() => {
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

onMounted(() => {
  fetchSetting()
})
</script>

<template>
  <n-card flex-1 rounded-10>
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
  </n-card>
</template>

<style scoped lang="scss"></style>
