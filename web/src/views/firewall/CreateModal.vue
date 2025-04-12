<script setup lang="ts">
import firewall from '@/api/panel/firewall'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const loading = ref(false)

const protocols = [
  {
    label: 'TCP',
    value: 'tcp'
  },
  {
    label: 'UDP',
    value: 'udp'
  },
  {
    label: 'TCP/UDP',
    value: 'tcp/udp'
  }
]

const families = [
  {
    label: 'IPv4',
    value: 'ipv4'
  },
  {
    label: 'IPv6',
    value: 'ipv6'
  }
]

const strategies = [
  {
    label: $gettext('Accept'),
    value: 'accept'
  },
  {
    label: $gettext('Drop'),
    value: 'drop'
  },
  {
    label: $gettext('Reject'),
    value: 'reject'
  }
]

const directions = [
  {
    label: $gettext('Inbound'),
    value: 'in'
  },
  {
    label: $gettext('Outbound'),
    value: 'out'
  }
]

const createModel = ref({
  family: 'ipv4',
  protocol: 'tcp',
  port_start: 80,
  port_end: 80,
  address: '',
  strategy: 'accept',
  direction: 'in'
})

const handleCreate = async () => {
  useRequest(firewall.createRule(createModel.value)).onSuccess(() => {
    show.value = false
    createModel.value = {
      family: 'ipv4',
      protocol: 'tcp',
      port_start: 80,
      port_end: 80,
      address: '',
      strategy: 'accept',
      direction: 'in'
    }
    window.$message.success($gettext('Created successfully'))
  })
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Create Rule')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="show = false"
  >
    <n-form :model="createModel">
      <n-form-item path="protocols" :label="$gettext('Transport Protocol')">
        <n-select v-model:value="createModel.protocol" :options="protocols" />
      </n-form-item>
      <n-form-item path="family" :label="$gettext('Network Protocol')">
        <n-select v-model:value="createModel.family" :options="families" />
      </n-form-item>
      <n-row :gutter="[0, 24]">
        <n-col :span="12">
          <n-form-item path="port_start" :label="$gettext('Start Port')">
            <n-input-number
              v-model:value="createModel.port_start"
              :min="1"
              :max="65535"
              placeholder="80"
            />
          </n-form-item>
        </n-col>
        <n-col :span="12">
          <n-form-item path="port_end" :label="$gettext('End Port')">
            <n-input-number
              v-model:value="createModel.port_end"
              :min="1"
              :max="65535"
              placeholder="80"
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item path="address" :label="$gettext('Target')">
        <n-input
          v-model:value="createModel.address"
          :placeholder="
            $gettext(
              'Optional IP or IP range: 127.0.0.1 or 172.16.0.0/24 (multiple separated by commas)'
            )
          "
        />
      </n-form-item>
      <n-form-item path="strategy" :label="$gettext('Strategy')">
        <n-select v-model:value="createModel.strategy" :options="strategies" />
      </n-form-item>
      <n-form-item path="strategy" :label="$gettext('Direction')">
        <n-select v-model:value="createModel.direction" :options="directions" />
      </n-form-item>
    </n-form>
    <n-button type="info" block :loading="loading" :disabled="loading" @click="handleCreate">
      {{ $gettext('Submit') }}
    </n-button>
  </n-modal>
</template>

<style scoped lang="scss"></style>
