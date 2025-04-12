<script setup lang="ts">
import container from '@/api/panel/container'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()

const props = defineProps({
  show: {
    type: Boolean,
    required: true
  }
})

const { show } = toRefs(props)
const doSubmit = ref(false)

const createModel = reactive({
  name: '',
  publish_all_ports: false,
  ports: [
    {
      container_start: 80,
      container_end: 80,
      host_start: 80,
      host_end: 80,
      host: '',
      protocol: 'tcp'
    }
  ],
  network: '',
  volumes: [
    {
      host: '/www',
      container: '/www',
      mode: 'rw'
    }
  ],
  cpus: 0,
  memory: 0,
  env: [],
  command: [],
  tty: false,
  restart_policy: 'no',
  labels: [],
  entrypoint: [],
  auto_remove: false,
  image: '',
  cpu_shares: 1024,
  privileged: false,
  open_stdin: false
})
const networks = ref<any>({})

const restartPolicyOptions = [
  { label: $gettext('None'), value: 'no' },
  { label: $gettext('Always'), value: 'always' },
  { label: $gettext('On failure (default 5 retries)'), value: 'on-failure' },
  { label: $gettext('Unless stopped'), value: 'unless-stopped' }
]

const addPortRow = () => {
  createModel.ports.push({
    container_start: 80,
    container_end: 80,
    host_start: 80,
    host_end: 80,
    host: '',
    protocol: 'tcp'
  })
}

const removePortRow = (index: number) => {
  createModel.ports.splice(index, 1)
}

const addVolumeRow = () => {
  createModel.volumes.push({
    host: '/www',
    container: '/www',
    mode: 'rw'
  })
}

const removeVolumeRow = (index: number) => {
  createModel.volumes.splice(index, 1)
}

const getNetworks = () => {
  useRequest(container.networkList(1, 1000)).onSuccess(({ data }) => {
    networks.value = data.items.map((item: any) => {
      return {
        label: item.name,
        value: item.id
      }
    })
    if (networks.value.length > 0) {
      createModel.network = networks.value[0].value
    }
  })
}

const handleSubmit = () => {
  doSubmit.value = true
  useRequest(container.containerCreate(createModel))
    .onSuccess(() => {
      window.$message.success($gettext('Created successfully'))
      handleClose()
    })
    .onComplete(() => {
      doSubmit.value = false
    })
}

const emit = defineEmits(['close'])

const handleClose = () => {
  emit('close')
}

onMounted(() => {
  getNetworks()
})
</script>

<template>
  <n-modal
    :title="$gettext('Create Container')"
    preset="card"
    style="width: 60vw"
    size="huge"
    :show="show"
    :bordered="false"
    :segmented="false"
    @close="handleClose"
  >
    <n-form :model="createModel">
      <n-form-item path="name" :label="$gettext('Container Name')">
        <n-input v-model:value="createModel.name" type="text" @keydown.enter.prevent />
      </n-form-item>
      <n-form-item path="name" :label="$gettext('Image')">
        <n-input v-model:value="createModel.image" type="text" @keydown.enter.prevent />
      </n-form-item>
      <n-form-item path="exposedAll" :label="$gettext('Ports')">
        <n-radio
          :checked="!createModel.publish_all_ports"
          :value="false"
          @change="createModel.publish_all_ports = !$event.target.value"
        >
          {{ $gettext('Map Ports') }}
        </n-radio>
        <n-radio
          :checked="createModel.publish_all_ports"
          :value="true"
          @change="createModel.publish_all_ports = !!$event.target.value"
        >
          {{ $gettext('Expose All') }}
        </n-radio>
      </n-form-item>
      <n-form-item path="ports" :label="$gettext('Port Mapping')" v-if="!createModel.publish_all_ports">
        <n-space vertical>
          <n-table striped>
            <thead>
              <tr>
                <th>IP</th>
                <th>{{ $gettext('Host (Start)') }}</th>
                <th>{{ $gettext('Host (End)') }}</th>
                <th>{{ $gettext('Container (Start)') }}</th>
                <th>{{ $gettext('Container (End)') }}</th>
                <th>{{ $gettext('Protocol') }}</th>
                <th>{{ $gettext('Actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(item, index) in createModel.ports" :key="index">
                <td>
                  <n-input
                    v-model:value="item.host"
                    type="text"
                    @keydown.enter.prevent
                    :placeholder="$gettext('Optional')"
                  />
                </td>
                <td>
                  <n-input-number
                    v-model:value="item.host_start"
                    type="text"
                    @keydown.enter.prevent
                  />
                </td>
                <td>
                  <n-input-number
                    v-model:value="item.host_end"
                    type="text"
                    @keydown.enter.prevent
                  />
                </td>
                <td>
                  <n-input-number
                    v-model:value="item.container_start"
                    type="text"
                    @keydown.enter.prevent
                  />
                </td>
                <td>
                  <n-input-number
                    v-model:value="item.container_end"
                    type="text"
                    @keydown.enter.prevent
                  />
                </td>
                <td>
                  <n-radio
                    :checked="item.protocol === 'tcp'"
                    value="tcp"
                    name="protocol"
                    @change="item.protocol = $event.target.value"
                  >
                    TCP
                  </n-radio>
                  <n-radio
                    :checked="item.protocol === 'udp'"
                    value="udp"
                    name="protocol"
                    @change="item.protocol = $event.target.value"
                  >
                    UDP
                  </n-radio>
                </td>
                <td><n-button @click="removePortRow(index)" size="small">{{ $gettext('Delete') }}</n-button></td>
              </tr>
            </tbody>
          </n-table>
          <n-button @click="addPortRow">{{ $gettext('Add') }}</n-button>
        </n-space>
      </n-form-item>
      <n-form-item path="network" :label="$gettext('Network')">
        <n-select v-model:value="createModel.network" :options="networks" />
      </n-form-item>
      <n-form-item path="mount" :label="$gettext('Mount')">
        <n-space vertical>
          <n-table striped>
            <thead>
              <tr>
                <th>{{ $gettext('Host Directory') }}</th>
                <th>{{ $gettext('Container Directory') }}</th>
                <th>{{ $gettext('Permission') }}</th>
                <th>{{ $gettext('Actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(item, index) in createModel.volumes" :key="index">
                <td>
                  <n-input v-model:value="item.host" type="text" @keydown.enter.prevent />
                </td>
                <td>
                  <n-input v-model:value="item.container" type="text" @keydown.enter.prevent />
                </td>
                <td>
                  <n-radio
                    :checked="item.mode === 'rw'"
                    value="rw"
                    name="mode"
                    @change="item.mode = $event.target.value"
                  >
                    {{ $gettext('Read-Write') }}
                  </n-radio>
                  <n-radio
                    :checked="item.mode === 'ro'"
                    value="ro"
                    name="mode"
                    @change="item.mode = $event.target.value"
                  >
                    {{ $gettext('Read-Only') }}
                  </n-radio>
                </td>
                <td><n-button @click="removeVolumeRow(index)" size="small">{{ $gettext('Delete') }}</n-button></td>
              </tr>
            </tbody>
          </n-table>
          <n-button @click="addVolumeRow">{{ $gettext('Add') }}</n-button>
        </n-space>
      </n-form-item>
      <n-form-item path="command" :label="$gettext('Command')">
        <n-dynamic-input v-model:value="createModel.command" :placeholder="$gettext('Command')" />
      </n-form-item>
      <n-form-item path="entrypoint" :label="$gettext('Entrypoint')">
        <n-dynamic-input v-model:value="createModel.entrypoint" :placeholder="$gettext('Entrypoint')" />
      </n-form-item>
      <n-row :gutter="[0, 24]">
        <n-col :span="8">
          <n-form-item path="memory" :label="$gettext('Memory')">
            <n-input-number v-model:value="createModel.memory" />
          </n-form-item>
        </n-col>
        <n-col :span="8">
          <n-form-item path="cpus" label="CPU">
            <n-input-number v-model:value="createModel.cpus" />
          </n-form-item>
        </n-col>
        <n-col :span="8">
          <n-form-item path="cpu_shares" :label="$gettext('CPU Shares')">
            <n-input-number v-model:value="createModel.cpu_shares" />
          </n-form-item>
        </n-col>
      </n-row>
      <n-row :gutter="[0, 24]">
        <n-col :span="6">
          <n-form-item path="tty" :label="$gettext('TTY (-t)')">
            <n-switch v-model:value="createModel.tty" />
          </n-form-item>
        </n-col>
        <n-col :span="6">
          <n-form-item path="open_stdin" :label="$gettext('STDIN (-i)')">
            <n-switch v-model:value="createModel.open_stdin" />
          </n-form-item>
        </n-col>
        <n-col :span="6">
          <n-form-item path="auto_remove" :label="$gettext('Auto Remove')">
            <n-switch v-model:value="createModel.auto_remove" />
          </n-form-item>
        </n-col>
        <n-col :span="6">
          <n-form-item path="privileged" :label="$gettext('Privileged Mode')">
            <n-switch v-model:value="createModel.privileged" />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item path="restart_policy" :label="$gettext('Restart Policy')">
        <n-select
          v-model:value="createModel.restart_policy"
          :placeholder="$gettext('Select restart policy')"
          :options="restartPolicyOptions"
        >
          {{ createModel.restart_policy || $gettext('Select restart policy') }}
        </n-select>
      </n-form-item>
      <n-form-item path="env" :label="$gettext('Environment Variables')">
        <n-dynamic-input
          v-model:value="createModel.env"
          preset="pair"
          :key-placeholder="$gettext('Variable Name')"
          :value-placeholder="$gettext('Variable Value')"
        />
      </n-form-item>
      <n-form-item path="labels" :label="$gettext('Labels')">
        <n-dynamic-input
          v-model:value="createModel.labels"
          preset="pair"
          :key-placeholder="$gettext('Label Name')"
          :value-placeholder="$gettext('Label Value')"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block :loading="doSubmit" :disabled="doSubmit" @click="handleSubmit">
      {{ $gettext('Submit') }}
    </n-button>
  </n-modal>
</template>

<style scoped lang="scss"></style>
